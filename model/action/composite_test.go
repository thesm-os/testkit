// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package action_test

import (
	"context"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/action"
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
