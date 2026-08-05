// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package action_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model"
	"go.thesmos.sh/testkit/engine/model/action"
)

func TestPool(t *testing.T) {
	t.Parallel()

	t.Run("matching get/put pass", func(t *testing.T) {
		t.Parallel()
		a := action.Pool(
			"Pool",
			func(_ context.Context, _ *simpleStore) (string, error) { return "r", nil },
			func(_ context.Context, _ *simpleStore, _ string) error { return nil },
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})

	t.Run("get error mismatch flagged", func(t *testing.T) {
		t.Parallel()
		i := 0
		a := action.Pool(
			"Pool",
			func(_ context.Context, _ *simpleStore) (string, error) {
				i++
				if i%2 == 1 {
					return "", errBroken
				}
				return "r", nil
			},
			func(_ context.Context, _ *simpleStore, _ string) error { return nil },
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err == nil {
				rt.Fatal("expected get-error mismatch")
			}
		})
	})
}

var errCursorBoom = errors.New("action: cursor boom")

func TestCursor(t *testing.T) {
	t.Parallel()

	t.Run("drains both sides equally", func(t *testing.T) {
		t.Parallel()
		nextCalls := 0
		a := action.Cursor(
			"Cursor",
			func(_ context.Context, _ *simpleStore) (string, bool, error) {
				nextCalls++
				if nextCalls%4 == 0 {
					return "", false, nil
				}
				return "v", true, nil
			},
			func(_ context.Context, _ *simpleStore) error { return nil },
			10,
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})

	// A Next that errors mid-drain must stop the drain and surface the
	// error rather than counting the failed call as exhaustion.
	t.Run("a Next error stops the drain on both sides", func(t *testing.T) {
		t.Parallel()
		a := action.Cursor(
			"Cursor",
			func(context.Context, *simpleStore) (string, bool, error) {
				return "", false, errCursorBoom
			},
			func(context.Context, *simpleStore) error { return nil },
			10,
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err == nil {
				rt.Fatal("a failing Next must not read as an exhausted cursor")
			}
		})
	})

	// A non-positive limit is a caller mistake, not a licence to iterate
	// forever — the action substitutes its own ceiling.
	t.Run("a non-positive limit falls back to the default ceiling", func(t *testing.T) {
		t.Parallel()
		a := action.Cursor(
			"Cursor",
			func(context.Context, *simpleStore) (string, bool, error) { return "v", true, nil },
			func(context.Context, *simpleStore) error { return nil },
			0,
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err == nil {
				rt.Fatal("an endless cursor must still hit the default limit")
			}
			if !strings.Contains(r.Err.Error(), "next limit 100") {
				rt.Fatalf("the diagnostic must name the substituted ceiling, got: %v", r.Err)
			}
		})
	})

	t.Run("infinite next hits the limit", func(t *testing.T) {
		t.Parallel()
		a := action.Cursor(
			"Cursor",
			func(_ context.Context, _ *simpleStore) (string, bool, error) { return "v", true, nil },
			func(_ context.Context, _ *simpleStore) error { return nil },
			3,
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err == nil {
				rt.Fatal("expected next-limit error")
			}
		})
	})
}

func TestTwoPhase(t *testing.T) {
	t.Parallel()

	t.Run("matching outcomes pass", func(t *testing.T) {
		t.Parallel()
		a := action.TwoPhase(
			"TwoPhase",
			func(_ context.Context, _ *simpleStore) (struct{}, error) { return struct{}{}, nil },
			func(_ context.Context, _ *simpleStore, _ struct{}) error { return nil },
			func(_ context.Context, _ *simpleStore, _ struct{}) error { return nil },
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})

	t.Run("begin error mismatch flagged", func(t *testing.T) {
		t.Parallel()
		i := 0
		a := action.TwoPhase(
			"TwoPhase",
			func(_ context.Context, _ *simpleStore) (struct{}, error) {
				i++
				if i%2 == 1 {
					return struct{}{}, errBroken
				}
				return struct{}{}, nil
			},
			func(_ context.Context, _ *simpleStore, _ struct{}) error { return nil },
			func(_ context.Context, _ *simpleStore, _ struct{}) error { return nil },
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err == nil {
				rt.Fatal("expected begin-error mismatch")
			}
		})
	})
}

func TestSaga(t *testing.T) {
	t.Parallel()

	t.Run("all steps succeed → no error", func(t *testing.T) {
		t.Parallel()
		a := action.Saga(
			"Order",
			[]func(context.Context, *simpleStore) error{
				func(_ context.Context, _ *simpleStore) error { return nil },
				func(_ context.Context, _ *simpleStore) error { return nil },
			},
			[]func(context.Context, *simpleStore) error{
				func(_ context.Context, _ *simpleStore) error { return nil },
				func(_ context.Context, _ *simpleStore) error { return nil },
			},
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})

	t.Run("step-failure index mismatch flagged", func(t *testing.T) {
		t.Parallel()
		// Distinguish SUT vs ref by identity: SUT carries no getF;
		// ref does. Step 0 fails iff the impl is the SUT marker.
		failOnSUT := func(_ context.Context, s *simpleStore) error {
			if s.getF == nil {
				return errBroken
			}
			return nil
		}
		ok := func(_ context.Context, _ *simpleStore) error { return nil }
		a := action.Saga(
			"Order",
			[]func(context.Context, *simpleStore) error{failOnSUT, ok},
			[]func(context.Context, *simpleStore) error{ok, ok},
		)
		sut := &simpleStore{}
		ref := &simpleStore{getF: func(_ string) (string, error) { return "", nil }}
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, sut, ref)
			if r.Err == nil {
				rt.Fatal("expected step-index mismatch")
			}
		})
	})
}

// Composite actions exist to catch a subject diverging from the reference.
// These drive the divergence arms directly: an oracle that agrees is the case
// the happy-path tests already cover, so what matters here is that a
// disagreement is reported and names the step it happened at.
func TestCompositeActionDivergence(t *testing.T) {
	t.Parallel()

	type box struct {
		getErr, putErr error
		items          int
		closeErr       error
		beginErr       error
		finErr         error
		failStep       int
	}
	// Fixtures are rebuilt per iteration: several of these actions mutate the
	// subject as they run (a cursor drains), and rapid invokes the property
	// many times, so a shared box would be exhausted after the first pass.
	// The first non-nil result wins, since a divergence on any iteration is
	// the thing under test.
	runFresh := func(t *testing.T, a model.Action[*box], mk func() (sut, ref *box)) model.ActionResult {
		t.Helper()
		var first model.ActionResult
		rapid.Check(t, func(rt *rapid.T) {
			sut, ref := mk()
			res := a.Run(rt, sut, ref)
			if first.Err == nil {
				first = res
			}
		})
		return first
	}
	run := func(t *testing.T, a model.Action[*box], sut, ref *box) model.ActionResult {
		t.Helper()
		return runFresh(t, a, func() (*box, *box) {
			s, r := *sut, *ref
			return &s, &r
		})
	}

	t.Run("Pool reports a Get divergence", func(t *testing.T) {
		t.Parallel()
		a := action.Pool("P",
			func(_ context.Context, b *box) (int, error) { return 0, b.getErr },
			func(_ context.Context, b *box, _ int) error { return b.putErr },
		)
		res := run(t, a, &box{getErr: errors.New("empty")}, &box{})
		if res.Err == nil || !strings.Contains(res.Err.Error(), "P.Get") {
			t.Fatalf("a Get disagreement must be reported, got: %v", res.Err)
		}
	})

	t.Run("Pool reports a Put divergence", func(t *testing.T) {
		t.Parallel()
		a := action.Pool("P",
			func(_ context.Context, b *box) (int, error) { return 0, nil },
			func(_ context.Context, b *box, _ int) error { return b.putErr },
		)
		res := run(t, a, &box{putErr: errors.New("full")}, &box{})
		if res.Err == nil || !strings.Contains(res.Err.Error(), "P.Put") {
			t.Fatalf("a Put disagreement must be reported, got: %v", res.Err)
		}
	})

	t.Run("Pool stops after a shared Get failure", func(t *testing.T) {
		t.Parallel()
		a := action.Pool("P",
			func(_ context.Context, b *box) (int, error) { return 0, b.getErr },
			func(_ context.Context, _ *box, _ int) error { t.Error("Put must not run"); return nil },
		)
		boom := errors.New("empty")
		if res := run(t, a, &box{getErr: boom}, &box{getErr: boom}); res.Err != nil {
			t.Fatalf("agreeing failures are not a divergence: %v", res.Err)
		}
	})

	newCursor := func(limit int) model.Action[*box] {
		return action.Cursor("C",
			func(_ context.Context, b *box) (int, bool, error) {
				if b.closeErr != nil && b.items < 0 {
					return 0, false, b.closeErr
				}
				if b.items == 0 {
					return 0, false, nil
				}
				b.items--
				return b.items, true, nil
			},
			func(_ context.Context, b *box) error { return b.closeErr },
			limit,
		)
	}

	t.Run("Cursor reports a yield-count divergence", func(t *testing.T) {
		t.Parallel()
		res := run(t, newCursor(100), &box{items: 3}, &box{items: 1})
		if res.Err == nil || !strings.Contains(res.Err.Error(), "C.Next") {
			t.Fatalf("differing yield counts must be reported, got: %v", res.Err)
		}
	})

	// A cursor that never reports exhaustion is a bug even when both sides
	// agree, so the limit overflow is a fault rather than a divergence.
	t.Run("Cursor reports a non-terminating drain", func(t *testing.T) {
		t.Parallel()
		res := run(t, newCursor(2), &box{items: 100}, &box{items: 100})
		if res.Err == nil || !strings.Contains(res.Err.Error(), "limit") {
			t.Fatalf("a drain that exceeds the limit must be reported, got: %v", res.Err)
		}
	})

	t.Run("Cursor reports a Close divergence", func(t *testing.T) {
		t.Parallel()
		res := run(t, newCursor(100), &box{closeErr: errors.New("busy")}, &box{})
		if res.Err == nil || !strings.Contains(res.Err.Error(), "C.Close") {
			t.Fatalf("a Close disagreement must be reported, got: %v", res.Err)
		}
	})

	newTwoPhase := func() model.Action[*box] {
		return action.TwoPhase("T",
			func(_ context.Context, b *box) (int, error) { return 1, b.beginErr },
			func(_ context.Context, b *box, _ int) error { return b.finErr },
			func(_ context.Context, b *box, _ int) error { return b.finErr },
		)
	}

	t.Run("TwoPhase reports a Begin divergence", func(t *testing.T) {
		t.Parallel()
		res := run(t, newTwoPhase(), &box{beginErr: errors.New("no tx")}, &box{})
		if res.Err == nil || !strings.Contains(res.Err.Error(), "T.Begin") {
			t.Fatalf("a Begin disagreement must be reported, got: %v", res.Err)
		}
	})

	t.Run("TwoPhase stops after a shared Begin failure", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("no tx")
		if res := run(t, newTwoPhase(), &box{beginErr: boom}, &box{beginErr: boom}); res.Err != nil {
			t.Fatalf("agreeing Begin failures are not a divergence: %v", res.Err)
		}
	})

	// The finaliser is chosen by a draw, so the diagnostic must name whichever
	// of Commit or Rollback actually ran.
	t.Run("TwoPhase names the finaliser that diverged", func(t *testing.T) {
		t.Parallel()
		res := run(t, newTwoPhase(), &box{finErr: errors.New("conflict")}, &box{})
		if res.Err == nil {
			t.Fatal("a finaliser disagreement must be reported")
		}
		msg := res.Err.Error()
		if !strings.Contains(msg, "T.Commit") && !strings.Contains(msg, "T.Rollback") {
			t.Fatalf("the diagnostic must name the finaliser, got: %s", msg)
		}
	})

	newSaga := func() model.Action[*box] {
		mkStep := func(i int) func(context.Context, *box) error {
			return func(_ context.Context, b *box) error {
				if b.failStep == i {
					return errors.New("step failed")
				}
				return nil
			}
		}
		return action.Saga("S",
			[]func(context.Context, *box) error{mkStep(0), mkStep(1), mkStep(2)},
			[]func(context.Context, *box) error{
				func(context.Context, *box) error { return nil },
				func(context.Context, *box) error { return nil },
				func(context.Context, *box) error { return nil },
			},
		)
	}

	t.Run("Saga reports a differing failure step", func(t *testing.T) {
		t.Parallel()
		res := run(t, newSaga(), &box{failStep: 1}, &box{failStep: 2})
		if res.Err == nil || !strings.Contains(res.Err.Error(), "failed at step") {
			t.Fatalf("a differing failure point must be reported, got: %v", res.Err)
		}
	})

	t.Run("Saga passes when both sides commit through", func(t *testing.T) {
		t.Parallel()
		if res := run(t, newSaga(), &box{failStep: -1}, &box{failStep: -1}); res.Err != nil {
			t.Fatalf("two clean runs are not a divergence: %v", res.Err)
		}
	})

	// A failing step rolls back the steps before it. Both sides fail at the
	// same point here, so the action reports no divergence — what this covers
	// is that compensation actually runs, including when a compensator itself
	// fails and the error is folded into the step error.
	t.Run("Saga compensates the steps before the failure", func(t *testing.T) {
		t.Parallel()
		compensated := 0
		a := action.Saga("S",
			[]func(context.Context, *box) error{
				func(context.Context, *box) error { return nil },
				func(context.Context, *box) error { return errors.New("step 1 failed") },
			},
			[]func(context.Context, *box) error{
				func(_ context.Context, b *box) error { compensated++; return b.putErr },
				func(context.Context, *box) error { return nil },
			},
		)
		if res := run(t, a, &box{putErr: errors.New("cannot undo")}, &box{}); res.Err != nil {
			t.Fatalf("both sides failing at the same step is not a divergence: %v", res.Err)
		}
		if compensated == 0 {
			t.Fatal("the step before the failure must be compensated")
		}
	})
}
