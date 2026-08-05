// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package fault_test

import (
	"errors"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/fault"
	"go.thesmos.sh/testkit/rand"
)

func TestCountedFault(t *testing.T) {
	t.Parallel()

	t.Run("fires on every Nth call", func(t *testing.T) {
		t.Parallel()
		f := fault.NewCountedFault[string](errors.New("boom"), 3)
		// Calls 1,2 → false; 3 → true; 4,5 → false; 6 → true
		results := make([]bool, 6)
		for i := range 6 {
			fired, _ := f.ShouldFire("ignored", nil)
			results[i] = fired
		}
		testkit.Equal(t, results, []bool{false, false, true, false, false, true},
			"must fire on 3rd and 6th call")
	})

	t.Run("returns configured error", func(t *testing.T) {
		t.Parallel()
		err := errors.New("injected")
		f := fault.NewCountedFault[string](err, 1)
		fired, got := f.ShouldFire("", nil)
		testkit.True(t, fired, "must fire on every call with n=1")
		testkit.ErrorIs(t, got, err, "must return configured error")
	})

	t.Run("disabled when n is zero", func(t *testing.T) {
		t.Parallel()
		f := fault.NewCountedFault[string](errors.New("boom"), 0)
		for range 10 {
			fired, _ := f.ShouldFire("", nil)
			testkit.False(t, fired, "must never fire when disabled")
		}
	})

	t.Run("disabled when n is negative", func(t *testing.T) {
		t.Parallel()
		f := fault.NewCountedFault[string](errors.New("boom"), -1)
		fired, _ := f.ShouldFire("", nil)
		testkit.False(t, fired, "must never fire with negative n")
	})

	t.Run("Reset rewinds counter", func(t *testing.T) {
		t.Parallel()
		f := fault.NewCountedFault[string](errors.New("boom"), 3)
		_, _ = f.ShouldFire("", nil) // call 1
		_, _ = f.ShouldFire("", nil) // call 2
		f.Reset()
		// After reset, fire should happen on 3rd call again.
		fired1, _ := f.ShouldFire("", nil) // call 1 after reset
		fired2, _ := f.ShouldFire("", nil) // call 2 after reset
		fired3, _ := f.ShouldFire("", nil) // call 3 after reset
		testkit.False(t, fired1, "call 1 after reset")
		testkit.False(t, fired2, "call 2 after reset")
		testkit.True(t, fired3, "call 3 after reset must fire")
	})

	t.Run("ignores call value and clock", func(t *testing.T) {
		t.Parallel()
		f := fault.NewCountedFault[int](errors.New("boom"), 1)
		// Different call values, nil clock — all fire the same.
		fired1, _ := f.ShouldFire(42, nil)
		fired2, _ := f.ShouldFire(0, nil)
		fired3, _ := f.ShouldFire(-1, clock.RealClock())
		testkit.True(t, fired1, "must fire regardless of call value")
		testkit.True(t, fired2, "must fire regardless of call value")
		testkit.True(t, fired3, "must fire regardless of clock")
	})
}

func TestRetryFault(t *testing.T) {
	t.Parallel()

	t.Run("fails first N-1 calls, succeeds on Nth", func(t *testing.T) {
		t.Parallel()
		f := fault.NewRetryFault[string](errors.New("unavailable"), 3)
		f1, e1 := f.ShouldFire("", nil) // call 1: fail
		f2, e2 := f.ShouldFire("", nil) // call 2: fail
		f3, _ := f.ShouldFire("", nil)  // call 3: succeed
		f4, _ := f.ShouldFire("", nil)  // call 4: still succeed

		testkit.True(t, f1, "call 1 must fail")
		testkit.Error(t, e1, "call 1 must return error")
		testkit.True(t, f2, "call 2 must fail")
		testkit.Error(t, e2, "call 2 must return error")
		testkit.False(t, f3, "call 3 must succeed")
		testkit.False(t, f4, "call 4 must succeed (finite fault)")
	})

	t.Run("n=1 succeeds immediately", func(t *testing.T) {
		t.Parallel()
		f := fault.NewRetryFault[string](errors.New("boom"), 1)
		fired, _ := f.ShouldFire("", nil)
		testkit.False(t, fired, "n=1 must succeed on first call")
	})

	t.Run("Reset rewinds counter", func(t *testing.T) {
		t.Parallel()
		f := fault.NewRetryFault[string](errors.New("boom"), 2)
		_, _ = f.ShouldFire("", nil) // call 1: fail
		_, _ = f.ShouldFire("", nil) // call 2: succeed
		f.Reset()
		f1, _ := f.ShouldFire("", nil) // call 1 after reset: fail again
		testkit.True(t, f1, "must fail again after Reset")
	})
}

func TestProbabilityFault(t *testing.T) {
	t.Parallel()

	t.Run("fires when random below threshold", func(t *testing.T) {
		t.Parallel()
		f := fault.NewProbabilityFault[string](errors.New("boom"), 0.5, rand.FixedRandSource(0.3))
		fired, err := f.ShouldFire("", nil)
		testkit.True(t, fired, "must fire when random < p")
		testkit.Error(t, err, "must return error")
	})

	t.Run("does not fire when random above threshold", func(t *testing.T) {
		t.Parallel()
		f := fault.NewProbabilityFault[string](errors.New("boom"), 0.5, rand.FixedRandSource(0.7))
		fired, _ := f.ShouldFire("", nil)
		testkit.False(t, fired, "must not fire when random >= p")
	})

	t.Run("p=1.0 always fires", func(t *testing.T) {
		t.Parallel()
		f := fault.NewProbabilityFault[string](errors.New("boom"), 1.0, rand.FixedRandSource(0.999))
		fired, _ := f.ShouldFire("", nil)
		testkit.True(t, fired, "p=1.0 must always fire")
	})

	t.Run("p=0.0 never fires", func(t *testing.T) {
		t.Parallel()
		f := fault.NewProbabilityFault[string](errors.New("boom"), 0.0, rand.FixedRandSource(0.0))
		fired, _ := f.ShouldFire("", nil)
		testkit.False(t, fired, "p=0.0 must never fire")
	})

	t.Run("Reset is a no-op", func(t *testing.T) {
		t.Parallel()
		f := fault.NewProbabilityFault[string](errors.New("boom"), 0.5, rand.FixedRandSource(0.3))
		f.Reset() // should not panic or change behavior
		fired, _ := f.ShouldFire("", nil)
		testkit.True(t, fired, "must still fire after Reset")
	})
}

func TestWindowedFault(t *testing.T) {
	t.Parallel()

	origin := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("fires before deadline", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		deadline := origin.Add(5 * time.Second)
		f := fault.NewWindowedFault[string](errors.New("down"), deadline)

		fired, err := f.ShouldFire("", clk)
		testkit.True(t, fired, "must fire before deadline")
		testkit.Error(t, err, "must return error")
	})

	t.Run("does not fire after deadline", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		deadline := origin.Add(5 * time.Second)
		f := fault.NewWindowedFault[string](errors.New("down"), deadline)

		clk.Advance(6 * time.Second)
		fired, _ := f.ShouldFire("", clk)
		testkit.False(t, fired, "must not fire after deadline")
	})

	t.Run("uses real time when clock is nil", func(t *testing.T) {
		t.Parallel()
		// Deadline in the far future — should fire.
		deadline := time.Now().Add(time.Hour)
		f := fault.NewWindowedFault[string](errors.New("down"), deadline)
		fired, _ := f.ShouldFire("", nil)
		testkit.True(t, fired, "must fire with nil clock and future deadline")
	})

	t.Run("does not fire with past deadline and nil clock", func(t *testing.T) {
		t.Parallel()
		deadline := time.Now().Add(-time.Hour)
		f := fault.NewWindowedFault[string](errors.New("down"), deadline)
		fired, _ := f.ShouldFire("", nil)
		testkit.False(t, fired, "must not fire with past deadline")
	})

	t.Run("Reset is a no-op", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		deadline := origin.Add(5 * time.Second)
		f := fault.NewWindowedFault[string](errors.New("down"), deadline)
		f.Reset() // should not change behavior
		fired, _ := f.ShouldFire("", clk)
		testkit.True(t, fired, "must still fire after Reset")
	})
}

func TestPredicateFault(t *testing.T) {
	t.Parallel()

	type call struct {
		RunID string
	}

	t.Run("fires when predicate matches", func(t *testing.T) {
		t.Parallel()
		f := fault.NewPredicateFault[call](
			errors.New("targeted"),
			func(c call) bool { return c.RunID == "run-1" },
		)
		fired, err := f.ShouldFire(call{RunID: "run-1"}, nil)
		testkit.True(t, fired, "must fire when predicate matches")
		testkit.Error(t, err, "must return error")
	})

	t.Run("does not fire when predicate rejects", func(t *testing.T) {
		t.Parallel()
		f := fault.NewPredicateFault[call](
			errors.New("targeted"),
			func(c call) bool { return c.RunID == "run-1" },
		)
		fired, _ := f.ShouldFire(call{RunID: "run-2"}, nil)
		testkit.False(t, fired, "must not fire when predicate rejects")
	})

	t.Run("Reset is a no-op", func(t *testing.T) {
		t.Parallel()
		f := fault.NewPredicateFault[call](
			errors.New("targeted"),
			func(c call) bool { return c.RunID == "run-1" },
		)
		f.Reset() // no-op
		fired, _ := f.ShouldFire(call{RunID: "run-1"}, nil)
		testkit.True(t, fired, "must still fire after Reset")
	})
}

func TestAndFault(t *testing.T) {
	t.Parallel()

	type call struct{ RunID string }

	t.Run("fires when all inner faults fire", func(t *testing.T) {
		t.Parallel()
		pred := fault.NewPredicateFault[call](errors.New("pred"), func(c call) bool { return c.RunID == "run-1" })
		counted := fault.NewCountedFault[call](errors.New("counted"), 1)
		composed := fault.And[call](pred, counted)

		fired, err := composed.ShouldFire(call{RunID: "run-1"}, nil)
		testkit.True(t, fired, "must fire when all match")
		testkit.Error(t, err, "must return error from first firing strategy")
	})

	t.Run("does not fire when one inner fault does not fire", func(t *testing.T) {
		t.Parallel()
		pred := fault.NewPredicateFault[call](errors.New("pred"), func(c call) bool { return c.RunID == "run-1" })
		counted := fault.NewCountedFault[call](errors.New("counted"), 1)
		composed := fault.And[call](pred, counted)

		fired, _ := composed.ShouldFire(call{RunID: "run-2"}, nil)
		testkit.False(t, fired, "must not fire when predicate rejects")
	})

	t.Run("Reset resets all inner faults", func(t *testing.T) {
		t.Parallel()
		counted := fault.NewCountedFault[call](errors.New("boom"), 2)
		composed := fault.And[call](counted)
		_, _ = composed.ShouldFire(call{}, nil) // call 1
		composed.Reset()
		// After reset, counter is back to 0. Next fire on call 2 again.
		f1, _ := composed.ShouldFire(call{}, nil) // call 1 after reset
		f2, _ := composed.ShouldFire(call{}, nil) // call 2 after reset
		testkit.False(t, f1, "call 1 after reset must not fire")
		testkit.True(t, f2, "call 2 after reset must fire")
	})
}

func TestOrFault(t *testing.T) {
	t.Parallel()

	type call struct{ RunID string }

	t.Run("fires when any inner fault fires", func(t *testing.T) {
		t.Parallel()
		errPred2 := errors.New("pred2")
		pred1 := fault.NewPredicateFault[call](errors.New("pred1"), func(c call) bool { return c.RunID == "run-1" })
		pred2 := fault.NewPredicateFault[call](errPred2, func(c call) bool { return c.RunID == "run-2" })
		composed := fault.Or[call](pred1, pred2)

		fired, err := composed.ShouldFire(call{RunID: "run-2"}, nil)
		testkit.True(t, fired, "must fire when second predicate matches")
		testkit.ErrorIs(t, err, errPred2, "must return error from firing strategy")
	})

	t.Run("does not fire when no inner fault fires", func(t *testing.T) {
		t.Parallel()
		pred := fault.NewPredicateFault[call](errors.New("pred"), func(c call) bool { return c.RunID == "run-1" })
		composed := fault.Or[call](pred)

		fired, _ := composed.ShouldFire(call{RunID: "run-99"}, nil)
		testkit.False(t, fired, "must not fire when no predicate matches")
	})

	t.Run("Reset resets all inner faults", func(t *testing.T) {
		t.Parallel()
		counted := fault.NewCountedFault[call](errors.New("boom"), 2)
		composed := fault.Or[call](counted)
		_, _ = composed.ShouldFire(call{}, nil) // call 1
		composed.Reset()
		f1, _ := composed.ShouldFire(call{}, nil)
		testkit.False(t, f1, "call 1 after reset must not fire")
	})
}

// Reset exists on every strategy so a caller can rewind a stub without knowing
// which strategy backs it. Only counter-based strategies have state to clear;
// the rest must accept the call and stay armed.
func TestResetIsANoOpForStatelessStrategies(t *testing.T) {
	t.Parallel()

	t.Run("probability fault stays armed", func(t *testing.T) {
		t.Parallel()
		f := fault.NewProbabilityFault[string](errors.New("boom"), 1.0, rand.FixedRandSource(0.999))
		f.Reset()
		fired, _ := f.ShouldFire("", nil)
		testkit.True(t, fired, "p=1.0 must still fire after Reset")
	})

	t.Run("windowed fault keeps its deadline", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(time.Unix(0, 0))
		f := fault.NewWindowedFault[string](errors.New("boom"), time.Unix(100, 0))
		f.Reset()
		fired, _ := f.ShouldFire("", clk)
		testkit.True(t, fired, "must still fire before the deadline after Reset")
	})

	t.Run("predicate fault keeps its predicate", func(t *testing.T) {
		t.Parallel()
		f := fault.NewPredicateFault(errors.New("boom"), func(v string) bool { return v == "hit" })
		f.Reset()
		fired, _ := f.ShouldFire("hit", nil)
		testkit.True(t, fired, "must still match the predicate after Reset")
	})
}
