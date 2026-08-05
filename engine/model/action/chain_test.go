// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package action_test

import (
	"context"
	"errors"
	"iter"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model/action"
	"go.thesmos.sh/testkit/engine/model/history"
	"go.thesmos.sh/testkit/engine/model/ref"
)

type entry struct {
	ID   string
	Data string
}

type chainIface interface {
	Append(context.Context, entry) error
	Replay(context.Context) iter.Seq2[entry, error]
	Verify(context.Context) error
}

// brokenChain always errors on Append and Verify.
type brokenChain struct {
	inner *ref.AppendOnly[entry]
}

func (*brokenChain) Append(_ context.Context, _ entry) error {
	return errors.New("injected error")
}

func (b *brokenChain) Replay(ctx context.Context) iter.Seq2[entry, error] {
	return b.inner.Replay(ctx)
}

func (*brokenChain) Verify(_ context.Context) error {
	return errors.New("injected verify error")
}

func TestChainAppendRecording(t *testing.T) {
	t.Parallel()

	t.Run("records successful appends to history", func(t *testing.T) {
		t.Parallel()
		hist := history.New[struct{}, entry]()
		entryGen := rapid.Just(entry{ID: "test", Data: "data"})

		a := action.ChainAppendRecording(
			"Append", entryGen, hist,
			func(_ entry) struct{} { return struct{}{} },
			func(ctx context.Context, impl chainIface, e entry) error {
				return impl.Append(ctx, e)
			},
		)

		sut := ref.NewAppendOnly[entry](nil)
		ref := ref.NewAppendOnly[entry](nil)

		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err != nil {
				rt.Fatalf("unexpected: %v", result.Err)
			}
		})

		if hist.TotalLen() == 0 {
			t.Fatal("history should have recorded appends")
		}
	})

	t.Run("catches SUT/ref error mismatch", func(t *testing.T) {
		t.Parallel()
		hist := history.New[struct{}, entry]()
		entryGen := rapid.Just(entry{ID: "mismatch", Data: "data"})

		a := action.ChainAppendRecording(
			"Append", entryGen, hist,
			func(_ entry) struct{} { return struct{}{} },
			func(ctx context.Context, impl chainIface, e entry) error {
				return impl.Append(ctx, e)
			},
		)

		sut := &brokenChain{inner: ref.NewAppendOnly[entry](nil)}
		ref := ref.NewAppendOnly[entry](nil)

		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err == nil {
				rt.Fatal("should have caught SUT/ref error mismatch on Append")
			}
		})
	})
}

func TestChainVerify(t *testing.T) {
	t.Parallel()

	t.Run("both valid chains pass", func(t *testing.T) {
		t.Parallel()
		a := action.ChainVerify(
			"Verify",
			func(ctx context.Context, impl chainIface) error {
				return impl.Verify(ctx)
			},
		)

		sut := ref.NewAppendOnly[entry](nil)
		ref := ref.NewAppendOnly[entry](nil)
		_ = sut.Append(t.Context(), entry{ID: "1"})
		_ = ref.Append(t.Context(), entry{ID: "1"})

		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err != nil {
				rt.Fatalf("unexpected: %v", result.Err)
			}
		})
	})

	t.Run("catches SUT/ref error mismatch", func(t *testing.T) {
		t.Parallel()
		a := action.ChainVerify(
			"Verify",
			func(ctx context.Context, impl chainIface) error {
				return impl.Verify(ctx)
			},
		)

		sut := &brokenChain{inner: ref.NewAppendOnly[entry](nil)}
		ref := ref.NewAppendOnly[entry](nil)

		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err == nil {
				rt.Fatal("should have caught SUT/ref error mismatch on Verify")
			}
		})
	})
}

func TestChainReplay(t *testing.T) {
	t.Parallel()

	t.Run("identical chains replay identically", func(t *testing.T) {
		t.Parallel()
		hist := history.New[struct{}, entry]()
		hist.Record(struct{}{}, entry{ID: "1"})

		a := action.ChainReplay(
			"Replay", hist,
			func(ctx context.Context, impl chainIface, _ struct{}) iter.Seq2[entry, error] {
				return impl.Replay(ctx)
			},
		)

		sut := ref.NewAppendOnly[entry](nil)
		ref := ref.NewAppendOnly[entry](nil)
		_ = sut.Append(t.Context(), entry{ID: "1"})
		_ = ref.Append(t.Context(), entry{ID: "1"})

		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err != nil {
				rt.Fatalf("unexpected: %v", result.Err)
			}
		})
	})

	t.Run("catches SUT/ref replay content mismatch", func(t *testing.T) {
		t.Parallel()
		hist := history.New[struct{}, entry]()
		hist.Record(struct{}{}, entry{ID: "1"})

		a := action.ChainReplay(
			"Replay", hist,
			func(ctx context.Context, impl chainIface, _ struct{}) iter.Seq2[entry, error] {
				return impl.Replay(ctx)
			},
		)

		// Ref has an entry, SUT doesn't -> content mismatch.
		sut := ref.NewAppendOnly[entry](nil)
		ref := ref.NewAppendOnly[entry](nil)
		_ = ref.Append(t.Context(), entry{ID: "1"})

		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err == nil {
				rt.Fatal("should have caught SUT/ref replay content mismatch")
			}
		})
	})
}

// ChainAppend and ChainReplay are differential actions: their job is to notice
// the subject and the reference disagreeing. Agreement — including agreeing to
// fail — is not a finding.
func TestChainActionDivergence(t *testing.T) {
	t.Parallel()

	type chainBox struct{ appendErr, replayErr error }

	t.Run("ChainAppend reports an append divergence", func(t *testing.T) {
		t.Parallel()
		a := action.ChainAppend("Append", rapid.Just("e"),
			func(_ context.Context, b *chainBox, _ string) error { return b.appendErr },
		)
		var got error
		rapid.Check(t, func(rt *rapid.T) {
			res := a.Run(rt, &chainBox{appendErr: errors.New("full")}, &chainBox{})
			if got == nil {
				got = res.Err
			}
		})
		if got == nil {
			t.Fatal("one side refusing an append is a divergence")
		}
	})

	t.Run("ChainAppend accepts agreement, including shared failure", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("full")
		a := action.ChainAppend("Append", rapid.Just("e"),
			func(_ context.Context, b *chainBox, _ string) error { return b.appendErr },
		)
		rapid.Check(t, func(rt *rapid.T) {
			if res := a.Run(rt, &chainBox{appendErr: boom}, &chainBox{appendErr: boom}); res.Err != nil {
				rt.Fatalf("both sides refusing is not a divergence: %v", res.Err)
			}
			if res := a.Run(rt, &chainBox{}, &chainBox{}); res.Err != nil {
				rt.Fatalf("both sides accepting is not a divergence: %v", res.Err)
			}
		})
	})

	// With nothing appended there is no partition to sample, so the action
	// must return cleanly rather than draw from an empty set.
	t.Run("ChainReplay is a no-op on an empty history", func(t *testing.T) {
		t.Parallel()
		hist := history.New[string, string]()
		a := action.ChainReplay("Replay", hist,
			func(_ context.Context, _ *chainBox, _ string) iter.Seq2[string, error] {
				t.Error("replay must not run without a partition")
				return func(func(string, error) bool) {}
			},
		)
		rapid.Check(t, func(rt *rapid.T) {
			if res := a.Run(rt, &chainBox{}, &chainBox{}); res.Err != nil {
				rt.Fatalf("an empty history has nothing to replay: %v", res.Err)
			}
		})
	})

	t.Run("ChainReplay reports a replay-error divergence", func(t *testing.T) {
		t.Parallel()
		hist := history.New[string, string]()
		hist.Record("p", "e")
		a := action.ChainReplay("Replay", hist,
			func(_ context.Context, b *chainBox, _ string) iter.Seq2[string, error] {
				return func(yield func(string, error) bool) {
					if b.replayErr != nil {
						yield("", b.replayErr)
						return
					}
					yield("e", nil)
				}
			},
		)
		var got error
		rapid.Check(t, func(rt *rapid.T) {
			res := a.Run(rt, &chainBox{replayErr: errors.New("io")}, &chainBox{})
			if got == nil {
				got = res.Err
			}
		})
		if got == nil {
			t.Fatal("one side failing to replay is a divergence")
		}
	})

	// drainSeq2 stops at the first error and returns what it had, so a
	// partially-yielded stream still reports its prefix alongside the failure.
	t.Run("ChainReplay reports differing contents", func(t *testing.T) {
		t.Parallel()
		hist := history.New[string, string]()
		hist.Record("p", "e")
		a := action.ChainReplay("Replay", hist,
			func(_ context.Context, b *chainBox, _ string) iter.Seq2[string, error] {
				return func(yield func(string, error) bool) {
					if b.replayErr != nil { // reused as a "differs" flag
						yield("different", nil)
						return
					}
					yield("e", nil)
				}
			},
		)
		var got error
		rapid.Check(t, func(rt *rapid.T) {
			res := a.Run(rt, &chainBox{replayErr: errors.New("x")}, &chainBox{})
			if got == nil {
				got = res.Err
			}
		})
		if got == nil {
			t.Fatal("differing replayed contents is a divergence")
		}
	})
}
