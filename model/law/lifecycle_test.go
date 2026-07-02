// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"context"
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

func TestLifecycleRespectsContext(t *testing.T) {
	t.Parallel()

	t.Run("op that checks context returns the cancellation error", func(t *testing.T) {
		t.Parallel()
		l := law.LifecycleRespectsContext[struct{}]{
			Op: func(ctx context.Context, _ struct{}) error {
				return ctx.Err()
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, struct{}{}, struct{}{}); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("op that ignores a cancelled context is caught", func(t *testing.T) {
		t.Parallel()
		l := law.LifecycleRespectsContext[struct{}]{
			Op: func(context.Context, struct{}) error { return nil }, // ignores ctx
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, struct{}{}, struct{}{}); err == nil {
				rt.Fatal("expected context-ignoring op to be caught")
			}
		})
	})
}

var errPoisoned = errors.New("poisoned")

// poisonBox reports a poison condition once set. When heals is set
// it (buggily) reverts to healthy after the first probe following
// the poison.
type poisonBox struct {
	poisoned bool
	heals    bool
	reads    int
}

func (b *poisonBox) poison() { b.poisoned = true }

func (b *poisonBox) probe() error {
	if !b.poisoned {
		return nil
	}
	if b.heals {
		b.reads++
		if b.reads > 1 {
			return nil // spontaneous healing — the bug
		}
	}
	return errPoisoned
}

func TestPoisonConsistent(t *testing.T) {
	t.Parallel()

	t.Run("sticky poison stays reported across reads", func(t *testing.T) {
		t.Parallel()
		l := law.PoisonConsistent[*poisonBox]{
			Poison: func(_ *rapid.T, b *poisonBox) { b.poison() },
			Probe:  func(_ *rapid.T, b *poisonBox) error { return b.probe() },
		}
		rapid.Check(t, func(rt *rapid.T) {
			b := &poisonBox{}
			if err := l.Check(rt, b, b); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("poison that spontaneously heals is caught", func(t *testing.T) {
		t.Parallel()
		l := law.PoisonConsistent[*poisonBox]{
			Poison: func(_ *rapid.T, b *poisonBox) { b.poison() },
			Probe:  func(_ *rapid.T, b *poisonBox) error { return b.probe() },
		}
		rapid.Check(t, func(rt *rapid.T) {
			b := &poisonBox{heals: true}
			if err := l.Check(rt, b, b); err == nil {
				rt.Fatal("expected spontaneous healing to be caught")
			}
		})
	})
}

var errStoreClosed = errors.New("store closed")

// closableStore rejects reads after Close — unless leaky is set, the
// bug where reads keep succeeding on a closed store.
type closableStore struct {
	closed bool
	leaky  bool
}

func (s *closableStore) close() error { s.closed = true; return nil }

func (s *closableStore) get() error {
	if s.closed && !s.leaky {
		return errStoreClosed
	}
	return nil
}

func TestLifecycleAfterCloseSentinel(t *testing.T) {
	t.Parallel()

	t.Run("closed store returns the sentinel from reads", func(t *testing.T) {
		t.Parallel()
		l := law.LifecycleAfterCloseSentinel[*closableStore]{
			Close:    func(_ *rapid.T, s *closableStore) error { return s.close() },
			Op:       func(_ *rapid.T, s *closableStore) error { return s.get() },
			Sentinel: errStoreClosed,
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := &closableStore{}
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("store that keeps serving after close is caught", func(t *testing.T) {
		t.Parallel()
		l := law.LifecycleAfterCloseSentinel[*closableStore]{
			Close:    func(_ *rapid.T, s *closableStore) error { return s.close() },
			Op:       func(_ *rapid.T, s *closableStore) error { return s.get() },
			Sentinel: errStoreClosed,
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := &closableStore{leaky: true}
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected post-close read to be caught")
			}
		})
	})
}
