// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"context"
	"iter"
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
