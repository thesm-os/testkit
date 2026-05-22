// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bmc_test

import (
	"errors"
	"strconv"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/model/bmc"
)

// counterState is the test fixture: a single int. The actions
// model Inc/Dec; the invariants assert non-negativity.
type counterState struct{ n int }

func incAction() bmc.Action[counterState] {
	return bmc.Action[counterState]{
		Name:  "Inc",
		Apply: func(s counterState) (counterState, error) { return counterState{n: s.n + 1}, nil },
	}
}

func decAction() bmc.Action[counterState] {
	return bmc.Action[counterState]{
		Name:  "Dec",
		Apply: func(s counterState) (counterState, error) { return counterState{n: s.n - 1}, nil },
	}
}

// guardedDecAction refuses to dec below zero — the safe variant.
func guardedDecAction() bmc.Action[counterState] {
	return bmc.Action[counterState]{
		Name: "Dec",
		Apply: func(s counterState) (counterState, error) {
			if s.n <= 0 {
				return s, bmc.ErrPreconditionUnmet
			}
			return counterState{n: s.n - 1}, nil
		},
	}
}

func nonNegative() bmc.Invariant[counterState] {
	return bmc.Invariant[counterState]{
		Name: "non-negative",
		Check: func(s counterState) error {
			if s.n < 0 {
				return bmc.Errorf("counter dropped below zero: %d", s.n)
			}
			return nil
		},
	}
}

func TestRunCertifiesSafety(t *testing.T) {
	t.Parallel()

	t.Run("guarded Dec never violates non-negativity", func(t *testing.T) {
		t.Parallel()
		out := bmc.Run(
			counterState{n: 0},
			[]bmc.Action[counterState]{incAction(), guardedDecAction()},
			[]bmc.Invariant[counterState]{nonNegative()},
			bmc.Config[counterState]{
				Depth:     6,
				StateHash: func(s counterState) string { return strconv.Itoa(s.n) },
			},
		)
		testkit.False(t, out.Violated(), "no violation expected")
		testkit.True(t, out.Explored > 1, "must have explored multiple states")
	})
}

func TestRunSurfacesPlantedBug(t *testing.T) {
	t.Parallel()

	t.Run("unguarded Dec produces a counterexample of length 1", func(t *testing.T) {
		t.Parallel()
		out := bmc.Run(
			counterState{n: 0},
			[]bmc.Action[counterState]{incAction(), decAction()},
			[]bmc.Invariant[counterState]{nonNegative()},
			bmc.Config[counterState]{
				Depth:     6,
				StateHash: func(s counterState) string { return strconv.Itoa(s.n) },
			},
		)
		testkit.True(t, out.Violated(), "violation expected")
		testkit.Equal(t, out.ViolatedInvariant, "non-negative", "invariant named")
		testkit.Equal(t, out.Counterexample, []string{"Dec"}, "shortest sequence")
		testkit.Equal(t, out.FailingState, counterState{n: -1}, "failing state captured")
		testkit.Assert(t, out.Reason).Contains("dropped below zero", "diagnostic preserved")
	})

	t.Run("initial-state violation surfaces with empty Counterexample", func(t *testing.T) {
		t.Parallel()
		out := bmc.Run(
			counterState{n: -1}, // already violating
			[]bmc.Action[counterState]{incAction()},
			[]bmc.Invariant[counterState]{nonNegative()},
			bmc.Config[counterState]{Depth: 4},
		)
		testkit.True(t, out.Violated(), "initial-state violation surfaced")
		testkit.Equal(t, len(out.Counterexample), 0, "no actions yet")
	})
}

func TestStateEquivalencePruning(t *testing.T) {
	t.Parallel()

	t.Run("hash dedup measurably reduces exploration", func(t *testing.T) {
		t.Parallel()
		actions := []bmc.Action[counterState]{incAction(), guardedDecAction()}
		invs := []bmc.Invariant[counterState]{nonNegative()}

		withHash := bmc.Run(
			counterState{n: 0}, actions, invs,
			bmc.Config[counterState]{
				Depth:     6,
				StateHash: func(s counterState) string { return strconv.Itoa(s.n) },
			},
		)
		withoutHash := bmc.Run(
			counterState{n: 0}, actions, invs,
			bmc.Config[counterState]{Depth: 6},
		)

		testkit.False(t, withHash.Violated(), "hash run safe")
		testkit.False(t, withoutHash.Violated(), "no-hash run safe")
		testkit.True(t, withHash.Pruned > 0, "pruning fired at least once")
		testkit.True(t, withHash.Explored < withoutHash.Explored,
			"pruned exploration smaller than un-pruned")
	})

	t.Run("nil StateHash explores without dedup", func(t *testing.T) {
		t.Parallel()
		out := bmc.Run(
			counterState{n: 0},
			[]bmc.Action[counterState]{incAction(), guardedDecAction()},
			[]bmc.Invariant[counterState]{nonNegative()},
			bmc.Config[counterState]{Depth: 4},
		)
		testkit.Equal(t, out.Pruned, 0, "no pruning without hash")
	})
}

func TestActionsThatSkip(t *testing.T) {
	t.Parallel()

	t.Run("Apply returning error skips that successor", func(t *testing.T) {
		t.Parallel()
		// "Skip" never produces a successor; only Inc does. At depth=3,
		// the engine should explore 0 → 1 → 2 → 3 only.
		skip := bmc.Action[counterState]{
			Name:  "Skip",
			Apply: func(s counterState) (counterState, error) { return s, errors.New("not applicable") },
		}
		out := bmc.Run(
			counterState{n: 0},
			[]bmc.Action[counterState]{incAction(), skip},
			nil,
			bmc.Config[counterState]{
				Depth:     3,
				StateHash: func(s counterState) string { return strconv.Itoa(s.n) },
			},
		)
		testkit.Equal(t, out.Explored, 4, "0,1,2,3 — Skip never advances")
		testkit.False(t, out.Violated(), "no invariants")
	})
}

func TestDepthZero(t *testing.T) {
	t.Parallel()

	t.Run("Depth=0 only checks the initial state", func(t *testing.T) {
		t.Parallel()
		out := bmc.Run(
			counterState{n: 0},
			[]bmc.Action[counterState]{incAction(), decAction()},
			[]bmc.Invariant[counterState]{nonNegative()},
			bmc.Config[counterState]{Depth: 0},
		)
		testkit.False(t, out.Violated(), "initial state safe")
		testkit.Equal(t, out.Explored, 1, "only initial counted")
	})

	t.Run("negative Depth treated as zero", func(t *testing.T) {
		t.Parallel()
		out := bmc.Run(
			counterState{n: 0},
			[]bmc.Action[counterState]{decAction()},
			[]bmc.Invariant[counterState]{nonNegative()},
			bmc.Config[counterState]{Depth: -5},
		)
		testkit.False(t, out.Violated(), "no exploration at negative depth")
	})
}

func TestCommandsBound(t *testing.T) {
	t.Parallel()

	t.Run("Commands caps the action set considered", func(t *testing.T) {
		t.Parallel()
		// Two actions; Commands=1 should only consider Inc, never Dec.
		// Without the cap, the unguarded-Dec planted bug surfaces;
		// with Commands=1, Dec is skipped entirely so no violation.
		out := bmc.Run(
			counterState{n: 0},
			[]bmc.Action[counterState]{incAction(), decAction()},
			[]bmc.Invariant[counterState]{nonNegative()},
			bmc.Config[counterState]{
				Depth:    4,
				Commands: 1,
				StateHash: func(s counterState) string {
					return strconv.Itoa(s.n)
				},
			},
		)
		testkit.False(t, out.Violated(), "Dec dropped by Commands=1")
		testkit.True(t, out.Explored >= 4, "Inc-only chain explored")
	})

	t.Run("Commands ≥ len(actions) is a no-op", func(t *testing.T) {
		t.Parallel()
		out := bmc.Run(
			counterState{n: 0},
			[]bmc.Action[counterState]{incAction(), decAction()},
			[]bmc.Invariant[counterState]{nonNegative()},
			bmc.Config[counterState]{Depth: 2, Commands: 99},
		)
		testkit.True(t, out.Violated(), "all actions still in play")
	})

	t.Run("Commands == 0 is a no-op (disabled)", func(t *testing.T) {
		t.Parallel()
		out := bmc.Run(
			counterState{n: 0},
			[]bmc.Action[counterState]{incAction(), decAction()},
			[]bmc.Invariant[counterState]{nonNegative()},
			bmc.Config[counterState]{Depth: 2, Commands: 0},
		)
		testkit.True(t, out.Violated(), "zero means disabled")
	})
}

func TestErrorf(t *testing.T) {
	t.Parallel()

	t.Run("formats with the invariant: prefix", func(t *testing.T) {
		t.Parallel()
		err := bmc.Errorf("explained: %d", 42)
		testkit.Assert(t, err.Error()).
			Contains("invariant:", "prefix").
			Contains("explained: 42", "formatted args")
	})
}
