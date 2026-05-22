// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"errors"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/law"
)

type lifecycleSUT struct {
	closed bool
	calls  int
}

func TestIdempotentLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("second call leaves Observe unchanged", func(t *testing.T) {
		t.Parallel()
		s := &lifecycleSUT{}
		l := law.IdempotentLifecycle[*lifecycleSUT, bool]{
			Call:    func(_ *rapid.T, c *lifecycleSUT) error { c.closed = true; return nil },
			Observe: func(_ *rapid.T, c *lifecycleSUT) bool { return c.closed },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("second call that mutates is flagged", func(t *testing.T) {
		t.Parallel()
		l := law.IdempotentLifecycle[*lifecycleSUT, int]{
			Call:    func(_ *rapid.T, c *lifecycleSUT) error { c.calls++; return nil },
			Observe: func(_ *rapid.T, c *lifecycleSUT) int { return c.calls },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &lifecycleSUT{}, &lifecycleSUT{}); err == nil {
				rt.Fatal("expected mutation flagged")
			}
		})
	})

	t.Run("second call that errors is flagged", func(t *testing.T) {
		t.Parallel()
		callN := 0
		l := law.IdempotentLifecycle[*lifecycleSUT, bool]{
			Call: func(_ *rapid.T, _ *lifecycleSUT) error {
				callN++
				if callN%2 == 0 {
					return errors.New("second-call error")
				}
				return nil
			},
			Observe: func(_ *rapid.T, _ *lifecycleSUT) bool { return false },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &lifecycleSUT{}, &lifecycleSUT{}); err == nil {
				rt.Fatal("expected second-call error to be flagged")
			}
		})
	})
}

func TestLeakFree(t *testing.T) {
	t.Parallel()

	t.Run("clean Open/Close cycle does not leak", func(t *testing.T) {
		t.Parallel()
		l := law.LeakFree[*lifecycleSUT]{
			Open:      func(_ *rapid.T, _ *lifecycleSUT) error { return nil },
			Close:     func(_ *rapid.T, _ *lifecycleSUT) error { return nil },
			Cycles:    8,
			Tolerance: 2,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &lifecycleSUT{}, &lifecycleSUT{}); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("erroring Open is vacuous", func(t *testing.T) {
		t.Parallel()
		l := law.LeakFree[*lifecycleSUT]{
			Open:   func(_ *rapid.T, _ *lifecycleSUT) error { return errors.New("nope") },
			Close:  func(_ *rapid.T, _ *lifecycleSUT) error { return nil },
			Cycles: 2,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &lifecycleSUT{}, &lifecycleSUT{}); err != nil {
				rt.Fatal(err)
			}
		})
	})
}

func TestPoisonNilOnFresh(t *testing.T) {
	t.Parallel()

	t.Run("fresh impl with nil probe passes", func(t *testing.T) {
		t.Parallel()
		l := law.PoisonNilOnFresh[*lifecycleSUT]{
			Factory: func() *lifecycleSUT { return &lifecycleSUT{} },
			Probe:   func(_ *rapid.T, _ *lifecycleSUT) error { return nil },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("fresh impl reporting poison is flagged", func(t *testing.T) {
		t.Parallel()
		l := law.PoisonNilOnFresh[*lifecycleSUT]{
			Factory: func() *lifecycleSUT { return &lifecycleSUT{} },
			Probe:   func(_ *rapid.T, _ *lifecycleSUT) error { return errors.New("poisoned at birth") },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); err == nil {
				rt.Fatal("expected fresh-poison to be flagged")
			}
		})
	})
}

func TestPoisonIdempotentRead(t *testing.T) {
	t.Parallel()

	t.Run("stable probe passes", func(t *testing.T) {
		t.Parallel()
		l := law.PoisonIdempotentRead[*lifecycleSUT]{
			Probe: func(_ *rapid.T, _ *lifecycleSUT) error { return nil },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &lifecycleSUT{}, &lifecycleSUT{}); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("toggling probe flagged", func(t *testing.T) {
		t.Parallel()
		call := 0
		l := law.PoisonIdempotentRead[*lifecycleSUT]{
			Probe: func(_ *rapid.T, _ *lifecycleSUT) error {
				call++
				if call%2 == 1 {
					return nil
				}
				return errors.New("now poisoned")
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &lifecycleSUT{}, &lifecycleSUT{}); err == nil {
				rt.Fatal("expected probe drift")
			}
		})
	})
}
