// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"context"
	"errors"
	"iter"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model/history"
	"go.thesmos.sh/testkit/engine/model/law"
	"go.thesmos.sh/testkit/engine/model/ref"
)

type entry struct {
	ID   string
	Data string
}

type chainImpl interface {
	Append(context.Context, entry) error
	Replay(context.Context) iter.Seq2[entry, error]
	Verify(context.Context) error
}

func replayFn(t *testing.T) func(*rapid.T, chainImpl, struct{}) iter.Seq2[entry, error] {
	t.Helper()
	return func(_ *rapid.T, impl chainImpl, _ struct{}) iter.Seq2[entry, error] {
		return impl.Replay(t.Context())
	}
}

// --- AppendOnlyHistoryGrows ---

func TestAppendOnlyHistoryGrows(t *testing.T) {
	t.Parallel()

	t.Run("passes for growing chain", func(t *testing.T) {
		t.Parallel()
		chain := ref.NewAppendOnly[entry](nil)
		hist := history.New[struct{}, entry]()
		_ = chain.Append(t.Context(), entry{ID: "1", Data: "a"})
		hist.Record(struct{}{}, entry{ID: "1", Data: "a"})

		l := &law.AppendOnlyHistoryGrows[chainImpl, struct{}, entry]{
			Replay: replayFn(t), Partitions: hist.Partitions,
		}

		rapid.Check(t, func(rt *rapid.T) {
			err := l.CheckWithStep(rt, chain, chain, 0)
			if err != nil {
				rt.Fatalf("should pass: %v", err)
			}
			_ = chain.Append(t.Context(), entry{ID: "2", Data: "b"})
			hist.Record(struct{}{}, entry{ID: "2", Data: "b"})
			err = l.CheckWithStep(rt, chain, chain, 1)
			if err != nil {
				rt.Fatalf("should pass after growth: %v", err)
			}
		})
	})

	t.Run("equal size is not shrinkage", func(t *testing.T) {
		t.Parallel()
		chain := ref.NewAppendOnly[entry](nil)
		hist := history.New[struct{}, entry]()
		_ = chain.Append(t.Context(), entry{ID: "1"})
		hist.Record(struct{}{}, entry{ID: "1"})

		l := &law.AppendOnlyHistoryGrows[chainImpl, struct{}, entry]{
			Replay: replayFn(t), Partitions: hist.Partitions,
		}

		rapid.Check(t, func(rt *rapid.T) {
			_ = l.CheckWithStep(rt, chain, chain, 0)
			err := l.CheckWithStep(rt, chain, chain, 1)
			if err != nil {
				rt.Fatalf("equal size should not be shrinkage: %v", err)
			}
		})
	})
}

// --- AppendOnlyNoDrops ---

func TestAppendOnlyNoDrops(t *testing.T) {
	t.Parallel()

	t.Run("passes when all appends present", func(t *testing.T) {
		t.Parallel()
		chain := ref.NewAppendOnly[entry](nil)
		hist := history.New[struct{}, entry]()
		e := entry{ID: "1", Data: "x"}
		_ = chain.Append(t.Context(), e)
		hist.Record(struct{}{}, e)

		l := law.AppendOnlyNoDrops[chainImpl, struct{}, entry]{
			Replay: replayFn(t), History: hist,
		}

		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, chain, chain)
			if err != nil {
				rt.Fatalf("should pass: %v", err)
			}
		})
	})

	t.Run("detects dropped entry", func(t *testing.T) {
		t.Parallel()
		chain := ref.NewAppendOnly[entry](nil)
		hist := history.New[struct{}, entry]()
		hist.Record(struct{}{}, entry{ID: "dropped"})

		l := law.AppendOnlyNoDrops[chainImpl, struct{}, entry]{
			Replay: replayFn(t), History: hist,
		}

		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, chain, chain)
			if err == nil {
				rt.Fatal("should detect dropped entry")
			}
		})
	})
}

// --- HashChainIntegrityViaVerify ---

func TestHashChainIntegrityViaVerify(t *testing.T) {
	t.Parallel()

	t.Run("passes for valid chain", func(t *testing.T) {
		t.Parallel()
		chain := ref.NewAppendOnly[entry](nil)
		_ = chain.Append(t.Context(), entry{ID: "1"})

		l := law.HashChainIntegrityViaVerify[chainImpl]{
			Verify: func(_ *rapid.T, impl chainImpl) error {
				return impl.Verify(t.Context())
			},
		}

		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, chain, chain)
			if err != nil {
				rt.Fatalf("should pass: %v", err)
			}
		})
	})

	t.Run("detects broken verify", func(t *testing.T) {
		t.Parallel()
		chain := ref.NewAppendOnly[entry](nil)
		_ = chain.Append(t.Context(), entry{ID: "1"})

		l := law.HashChainIntegrityViaVerify[chainImpl]{
			Verify: func(_ *rapid.T, _ chainImpl) error {
				return ref.ErrChainIntegrity
			},
		}

		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, chain, chain)
			if err == nil {
				rt.Fatal("should detect broken verify")
			}
		})
	})
}

// --- ReplayDeterminism ---

func TestReplayDeterminism(t *testing.T) {
	t.Parallel()

	t.Run("passes for deterministic chain", func(t *testing.T) {
		t.Parallel()
		chain := ref.NewAppendOnly[entry](nil)
		hist := history.New[struct{}, entry]()
		_ = chain.Append(t.Context(), entry{ID: "1"})
		_ = chain.Append(t.Context(), entry{ID: "2"})
		hist.Record(struct{}{}, entry{ID: "1"})
		hist.Record(struct{}{}, entry{ID: "2"})

		l := law.ReplayDeterminism[chainImpl, struct{}, entry]{
			Replay: replayFn(t), Partitions: hist.Partitions,
		}

		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, chain, chain)
			if err != nil {
				rt.Fatalf("should pass: %v", err)
			}
		})
	})

	t.Run("detects nondeterministic replay", func(t *testing.T) {
		t.Parallel()
		hist := history.New[struct{}, entry]()
		hist.Record(struct{}{}, entry{ID: "1"})
		callCount := 0

		l := law.ReplayDeterminism[chainImpl, struct{}, entry]{
			Replay: func(_ *rapid.T, _ chainImpl, _ struct{}) iter.Seq2[entry, error] {
				callCount++
				if callCount%2 == 0 {
					return func(yield func(entry, error) bool) {
						yield(entry{ID: "flipped"}, nil)
					}
				}
				return func(yield func(entry, error) bool) {
					yield(entry{ID: "1"}, nil)
				}
			},
			Partitions: hist.Partitions,
		}

		chain := ref.NewAppendOnly[entry](nil)
		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, chain, chain)
			if err == nil {
				rt.Fatal("should detect nondeterministic replay")
			}
		})
	})
}

// --- ReplayRespectsCausality ---

func TestReplayRespectsCausality(t *testing.T) {
	t.Parallel()

	t.Run("passes for correctly ordered chain", func(t *testing.T) {
		t.Parallel()
		chain := ref.NewAppendOnly[entry](nil)
		hist := history.New[struct{}, entry]()
		_ = chain.Append(t.Context(), entry{ID: "1"})
		_ = chain.Append(t.Context(), entry{ID: "2"})
		hist.Record(struct{}{}, entry{ID: "1"})
		hist.Record(struct{}{}, entry{ID: "2"})

		l := law.ReplayRespectsCausality[chainImpl, struct{}, entry]{
			Replay:     replayFn(t),
			Partitions: hist.Partitions,
			EntryID:    func(e entry) string { return e.ID },
			DependsOn: func(e entry) []string {
				if e.ID == "2" {
					return []string{"1"}
				}
				return nil
			},
		}

		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, chain, chain)
			if err != nil {
				rt.Fatalf("should pass: %v", err)
			}
		})
	})

	t.Run("detects missing dependency in replay", func(t *testing.T) {
		t.Parallel()
		hist := history.New[struct{}, entry]()
		hist.Record(struct{}{}, entry{ID: "1"})

		l := law.ReplayRespectsCausality[chainImpl, struct{}, entry]{
			Replay: func(_ *rapid.T, _ chainImpl, _ struct{}) iter.Seq2[entry, error] {
				return func(yield func(entry, error) bool) {
					yield(entry{ID: "1"}, nil)
				}
			},
			Partitions: hist.Partitions,
			EntryID:    func(e entry) string { return e.ID },
			DependsOn: func(e entry) []string {
				if e.ID == "1" {
					return []string{"0"}
				}
				return nil
			},
		}

		chain := ref.NewAppendOnly[entry](nil)
		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, chain, chain)
			if err == nil {
				rt.Fatal("should detect missing dependency")
			}
		})
	})
}

// HashChainIntegrityViaErr reads a poison flag off both subject and reference.
// It must name which side is poisoned — a report that says only "integrity
// violation" leaves the reader unable to tell a broken subject from a broken
// oracle.
func TestHashChainIntegrityViaErr(t *testing.T) {
	t.Parallel()

	type poisonable struct{ err error }
	read := func(p *poisonable) error { return p.err }

	t.Run("both clean passes", func(t *testing.T) {
		t.Parallel()
		l := law.HashChainIntegrityViaErr[*poisonable]{Err: read}
		if err := l.Check(nil, &poisonable{}, &poisonable{}); err != nil {
			t.Fatalf("two clean chains must pass: %v", err)
		}
	})

	t.Run("a poisoned SUT is named", func(t *testing.T) {
		t.Parallel()
		l := law.HashChainIntegrityViaErr[*poisonable]{Err: read}
		err := l.Check(nil, &poisonable{err: errors.New("corrupt")}, &poisonable{})
		if err == nil {
			t.Fatal("a poisoned SUT is a violation")
		}
		if !strings.Contains(err.Error(), "SUT") {
			t.Fatalf("the diagnostic must identify the SUT, got: %v", err)
		}
	})

	t.Run("a poisoned reference is named", func(t *testing.T) {
		t.Parallel()
		l := law.HashChainIntegrityViaErr[*poisonable]{Err: read}
		err := l.Check(nil, &poisonable{}, &poisonable{err: errors.New("corrupt")})
		if err == nil {
			t.Fatal("a poisoned reference is a violation")
		}
		if !strings.Contains(err.Error(), "ref") {
			t.Fatalf("the diagnostic must identify the reference, got: %v", err)
		}
	})

	t.Run("identity is distinct from the Verify variant", func(t *testing.T) {
		t.Parallel()
		var viaErr law.HashChainIntegrityViaErr[*poisonable]
		var viaVerify law.HashChainIntegrityViaVerify[*poisonable]
		if viaErr.ID() == viaVerify.ID() {
			t.Fatalf("the two hash-chain laws must not share an ID (%q)", viaErr.ID())
		}
	})
}

// A replay that errors mid-stream is a fault for every chain law: the laws
// compare what the chain hands back, so a partial stream makes the comparison
// meaningless rather than merely unfavourable. drain stops at the first error
// and returns the prefix it had, so each law must surface it rather than
// judging a truncated replay.
func TestChainLawsSurfaceReplayErrors(t *testing.T) {
	t.Parallel()

	boom := errors.New("replay failed")
	failing := func(*rapid.T, int, string) iter.Seq2[string, error] {
		return func(yield func(string, error) bool) { yield("", boom) }
	}
	parts := func() []string { return []string{"p"} }

	t.Run("AppendOnlyHistoryGrows", func(t *testing.T) {
		t.Parallel()
		l := &law.AppendOnlyHistoryGrows[int, string, string]{
			Replay: failing, Partitions: parts,
		}
		if err := l.Check(nil, 0, 0); err == nil {
			t.Fatal("a failing replay must be surfaced")
		}
	})

	t.Run("ReplayDeterminism", func(t *testing.T) {
		t.Parallel()
		l := law.ReplayDeterminism[int, string, string]{Replay: failing, Partitions: parts}
		if err := l.Check(nil, 0, 0); err == nil {
			t.Fatal("a failing first replay must be surfaced")
		}
	})

	t.Run("ReplayRespectsCausality", func(t *testing.T) {
		t.Parallel()
		l := law.ReplayRespectsCausality[int, string, string]{
			Replay: failing, Partitions: parts,
			DependsOn: func(string) []string { return nil },
		}
		if err := l.Check(nil, 0, 0); err == nil {
			t.Fatal("a failing replay must be surfaced")
		}
	})

	// Determinism needs two replays, and the second one failing is a
	// different defect from the first — the chain answered once already.
	t.Run("ReplayDeterminism surfaces a second replay that fails", func(t *testing.T) {
		t.Parallel()
		replays := 0
		l := law.ReplayDeterminism[int, string, string]{
			Partitions: parts,
			Replay: func(*rapid.T, int, string) iter.Seq2[string, error] {
				replays++
				n := replays
				return func(yield func(string, error) bool) {
					if n > 1 {
						yield("", boom)
						return
					}
					yield("a", nil)
				}
			},
		}
		err := l.Check(nil, 0, 0)
		if err == nil || !strings.Contains(err.Error(), "second replay error") {
			t.Fatalf("a chain that stops replaying must be reported, got: %v", err)
		}
	})

	t.Run("ReplayDeterminism flags replays of differing length", func(t *testing.T) {
		t.Parallel()
		replays := 0
		l := law.ReplayDeterminism[int, string, string]{
			Partitions: parts,
			Replay: func(*rapid.T, int, string) iter.Seq2[string, error] {
				replays++
				short := replays > 1
				return func(yield func(string, error) bool) {
					if !yield("a", nil) || short {
						return
					}
					yield("b", nil)
				}
			},
		}
		err := l.Check(nil, 0, 0)
		if err == nil || !strings.Contains(err.Error(), "first has 2, second has 1") {
			t.Fatalf("a replay that loses entries must be reported, got: %v", err)
		}
	})
}

// The growth law is stateful: it remembers the prior replay per partition and
// compares each new one against it. Step 0 resets that memory, which is what
// lets the same law instance be reused across rapid iterations.
func TestAppendOnlyHistoryGrowsStateTransitions(t *testing.T) {
	t.Parallel()

	newLaw := func(seq *[][]string) *law.AppendOnlyHistoryGrows[int, string, string] {
		i := 0
		return &law.AppendOnlyHistoryGrows[int, string, string]{
			Partitions: func() []string { return []string{"p"} },
			Replay: func(*rapid.T, int, string) iter.Seq2[string, error] {
				cur := (*seq)[min(i, len(*seq)-1)]
				i++
				return func(yield func(string, error) bool) {
					for _, e := range cur {
						if !yield(e, nil) {
							return
						}
					}
				}
			},
		}
	}

	t.Run("a growing chain passes", func(t *testing.T) {
		t.Parallel()
		seq := [][]string{{"a"}, {"a", "b"}}
		l := newLaw(&seq)
		if err := l.Check(nil, 0, 0); err != nil {
			t.Fatalf("the first observation cannot fail: %v", err)
		}
		if err := l.Check(nil, 0, 0); err != nil {
			t.Fatalf("an extended chain must pass: %v", err)
		}
	})

	t.Run("a shrinking chain is a violation", func(t *testing.T) {
		t.Parallel()
		seq := [][]string{{"a", "b"}, {"a"}}
		l := newLaw(&seq)
		_ = l.Check(nil, 0, 0)
		err := l.Check(nil, 0, 0)
		if err == nil {
			t.Fatal("a chain that loses entries is a violation")
		}
		if !strings.Contains(err.Error(), "shrank") {
			t.Fatalf("the diagnostic must name the shrink, got: %v", err)
		}
	})

	t.Run("a mutated entry is a violation", func(t *testing.T) {
		t.Parallel()
		seq := [][]string{{"a", "b"}, {"a", "CHANGED"}}
		l := newLaw(&seq)
		_ = l.Check(nil, 0, 0)
		err := l.Check(nil, 0, 0)
		if err == nil {
			t.Fatal("rewriting a committed entry is a violation")
		}
		if !strings.Contains(err.Error(), "mutated") {
			t.Fatalf("the diagnostic must name the mutation, got: %v", err)
		}
	})

	// Step 0 marks a fresh run, so prior state from the previous run must not
	// leak into it and produce a phantom shrink.
	t.Run("step zero resets the remembered chain", func(t *testing.T) {
		t.Parallel()
		seq := [][]string{{"a", "b"}, {"a"}}
		l := newLaw(&seq)
		_ = l.CheckWithStep(nil, 0, 0, 0)
		if err := l.CheckWithStep(nil, 0, 0, 0); err != nil {
			t.Fatalf("a reset must forget the longer prior chain: %v", err)
		}
	})
}

// Like its Err-based sibling, the Verify-based integrity law must say which
// side failed — a bare "integrity violation" cannot distinguish a broken
// subject from a broken oracle.
func TestHashChainIntegrityViaVerifyNamesTheSide(t *testing.T) {
	t.Parallel()

	type box struct{ err error }
	verify := func(_ *rapid.T, b *box) error { return b.err }
	l := law.HashChainIntegrityViaVerify[*box]{Verify: verify}

	t.Run("both intact passes", func(t *testing.T) {
		t.Parallel()
		if err := l.Check(nil, &box{}, &box{}); err != nil {
			t.Fatalf("two intact chains must pass: %v", err)
		}
	})

	t.Run("a failing SUT is named", func(t *testing.T) {
		t.Parallel()
		err := l.Check(nil, &box{err: errors.New("corrupt")}, &box{})
		if err == nil || !strings.Contains(err.Error(), "SUT") {
			t.Fatalf("the diagnostic must identify the SUT, got: %v", err)
		}
	})

	t.Run("a failing reference is named", func(t *testing.T) {
		t.Parallel()
		err := l.Check(nil, &box{}, &box{err: errors.New("corrupt")})
		if err == nil || !strings.Contains(err.Error(), "ref") {
			t.Fatalf("the diagnostic must identify the reference, got: %v", err)
		}
	})
}

// AppendOnlyNoDrops is a membership check against the recorded history: it
// catches a chain that accepted a write and then silently lost it, which pure
// replay-vs-replay comparison cannot see because the SUT owns the read surface.
func TestAppendOnlyNoDropsBranches(t *testing.T) {
	t.Parallel()

	replayOf := func(items map[string][]string) func(*rapid.T, int, string) iter.Seq2[string, error] {
		return func(_ *rapid.T, _ int, k string) iter.Seq2[string, error] {
			return func(yield func(string, error) bool) {
				for _, e := range items[k] {
					if !yield(e, nil) {
						return
					}
				}
			}
		}
	}

	t.Run("a chain that replays everything passes", func(t *testing.T) {
		t.Parallel()
		h := history.New[string, string]()
		h.Record("p", "a")
		h.Record("p", "b")
		l := law.AppendOnlyNoDrops[int, string, string]{
			History: h, Replay: replayOf(map[string][]string{"p": {"a", "b"}}),
		}
		if err := l.Check(nil, 0, 0); err != nil {
			t.Fatalf("no drops means no violation: %v", err)
		}
	})

	t.Run("a silently dropped append is a violation", func(t *testing.T) {
		t.Parallel()
		h := history.New[string, string]()
		h.Record("p", "a")
		h.Record("p", "b")
		l := law.AppendOnlyNoDrops[int, string, string]{
			History: h, Replay: replayOf(map[string][]string{"p": {"a"}}),
		}
		err := l.Check(nil, 0, 0)
		if err == nil {
			t.Fatal("an accepted append missing from replay is a drop")
		}
		if !strings.Contains(err.Error(), "missing from replay") {
			t.Fatalf("the diagnostic must name the drop, got: %v", err)
		}
	})

	// Membership, not multiplicity: a chain that dedupes a retried append has
	// not dropped anything, so the law must not fire.
	t.Run("deduplication is not a drop", func(t *testing.T) {
		t.Parallel()
		h := history.New[string, string]()
		h.Record("p", "a")
		h.Record("p", "a")
		l := law.AppendOnlyNoDrops[int, string, string]{
			History: h, Replay: replayOf(map[string][]string{"p": {"a"}}),
		}
		if err := l.Check(nil, 0, 0); err != nil {
			t.Fatalf("a deduped retry is not a dropped write: %v", err)
		}
	})

	t.Run("a partition with no attempts is skipped", func(t *testing.T) {
		t.Parallel()
		h := history.New[string, string]()
		h.Record("p", "a")
		l := law.AppendOnlyNoDrops[int, string, string]{
			History: h, Replay: replayOf(map[string][]string{"p": {"a"}}),
		}
		if err := l.Check(nil, 0, 0); err != nil {
			t.Fatalf("an empty partition contributes nothing: %v", err)
		}
	})

	t.Run("a failing replay is surfaced", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("replay failed")
		h := history.New[string, string]()
		h.Record("p", "a")
		l := law.AppendOnlyNoDrops[int, string, string]{
			History: h,
			Replay: func(*rapid.T, int, string) iter.Seq2[string, error] {
				return func(yield func(string, error) bool) { yield("", boom) }
			},
		}
		err := l.Check(nil, 0, 0)
		if err == nil || !strings.Contains(err.Error(), "replay error") {
			t.Fatalf("a chain that cannot be replayed must be reported, got: %v", err)
		}
	})
}
