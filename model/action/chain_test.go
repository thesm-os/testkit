// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package action_test

import (
	"context"
	"errors"
	"iter"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/action"
	"go.thesmos.sh/testkit/model/history"
	"go.thesmos.sh/testkit/model/refchain"
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
	inner *refchain.AppendOnly[entry]
}

func (b *brokenChain) Append(_ context.Context, _ entry) error {
	return errors.New("injected error")
}

func (b *brokenChain) Replay(ctx context.Context) iter.Seq2[entry, error] {
	return b.inner.Replay(ctx)
}

func (b *brokenChain) Verify(_ context.Context) error {
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

		sut := refchain.New[entry](nil)
		ref := refchain.New[entry](nil)

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

		sut := &brokenChain{inner: refchain.New[entry](nil)}
		ref := refchain.New[entry](nil)

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
		a := action.ChainVerify("Verify",
			func(ctx context.Context, impl chainIface) error {
				return impl.Verify(ctx)
			},
		)

		sut := refchain.New[entry](nil)
		ref := refchain.New[entry](nil)
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
		a := action.ChainVerify("Verify",
			func(ctx context.Context, impl chainIface) error {
				return impl.Verify(ctx)
			},
		)

		sut := &brokenChain{inner: refchain.New[entry](nil)}
		ref := refchain.New[entry](nil)

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

		a := action.ChainReplay("Replay", hist,
			func(ctx context.Context, impl chainIface, _ struct{}) iter.Seq2[entry, error] {
				return impl.Replay(ctx)
			},
		)

		sut := refchain.New[entry](nil)
		ref := refchain.New[entry](nil)
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

		a := action.ChainReplay("Replay", hist,
			func(ctx context.Context, impl chainIface, _ struct{}) iter.Seq2[entry, error] {
				return impl.Replay(ctx)
			},
		)

		// Ref has an entry, SUT doesn't -> content mismatch.
		sut := refchain.New[entry](nil)
		ref := refchain.New[entry](nil)
		_ = ref.Append(t.Context(), entry{ID: "1"})

		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err == nil {
				rt.Fatal("should have caught SUT/ref replay content mismatch")
			}
		})
	})
}
