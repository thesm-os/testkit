// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model/law"
)

type lifecycleSUT struct {
	closed bool
	calls  int
}

func TestIdempotentLifecycle(t *testing.T) {
	t.Parallel()

	// The marker is the runner's dispatch, on the value receiver: a registry
	// holding the law by value must still route it to a throwaway pair.
	var iso law.Isolated = law.IdempotentLifecycle[*lifecycleSUT, bool]{}
	iso.IsolatedLaw()

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

	// LeakFree samples runtime.NumGoroutine() either side of the cycle, so its
	// verdict reflects the whole process, not just the subject. Asserting that
	// verdict makes the test depend on whatever else the suite is running
	// concurrently, which is why this checks the mechanism — every cycle drives
	// Open and Close — rather than the goroutine count.
	t.Run("a clean cycle drives Open and Close the configured number of times", func(t *testing.T) {
		t.Parallel()
		var opens, closes int
		l := law.LeakFree[*lifecycleSUT]{
			Open:      func(_ *rapid.T, _ *lifecycleSUT) error { opens++; return nil },
			Close:     func(_ *rapid.T, _ *lifecycleSUT) error { closes++; return nil },
			Cycles:    8,
			Tolerance: 2,
		}
		rapid.Check(t, func(rt *rapid.T) {
			opens, closes = 0, 0
			_ = l.Check(rt, &lifecycleSUT{}, &lifecycleSUT{})
			if opens != 8 || closes != 8 {
				rt.Fatalf("expected 8 balanced cycles, got %d opens / %d closes", opens, closes)
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
			if err := l.Check(rt, &lifecycleSUT{}, &lifecycleSUT{}); !law.Holds(err) {
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

	// The marker is the runner's dispatch, on the value receiver: a registry
	// holding the law by value must still route it to a throwaway pair.
	var iso law.Isolated = law.PoisonConsistent[*poisonBox]{}
	iso.IsolatedLaw()

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

	// The marker is the runner's dispatch, on the value receiver: a registry
	// holding the law by value must still route it to a throwaway pair.
	var iso law.Isolated = law.LifecycleAfterCloseSentinel[*closableStore]{}
	iso.IsolatedLaw()

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

// The lifecycle laws share one shape: a setup call that may be refused (a
// precondition, so the law holds vacuously) and a follow-up whose behaviour is
// the actual subject. These drive both sides plus the tunables' defaults.
func TestLifecycleLawBranches(t *testing.T) {
	t.Parallel()

	t.Run("IdempotentLifecycle holds vacuously when the first call is refused", func(t *testing.T) {
		t.Parallel()
		l := law.IdempotentLifecycle[int, int]{
			Call:    func(*rapid.T, int) error { return errors.New("closed") },
			Observe: func(*rapid.T, int) int { return 0 },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); !law.Holds(err) {
				rt.Fatalf("a refused first call is a precondition: %v", err)
			}
		})
	})

	t.Run("IdempotentLifecycle flags a second call that errors", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			calls := 0
			l := law.IdempotentLifecycle[int, int]{
				Call: func(*rapid.T, int) error {
					calls++
					if calls > 1 {
						return errors.New("already open")
					}
					return nil
				},
				Observe: func(*rapid.T, int) int { return 0 },
			}
			if err := l.Check(rt, 0, 0); err == nil {
				rt.Fatal("a lifecycle that rejects a repeat call is not idempotent")
			}
		})
	})

	t.Run("IdempotentLifecycle flags a second call that mutates state", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			n := 0
			l := law.IdempotentLifecycle[int, int]{
				Call:    func(*rapid.T, int) error { n++; return nil },
				Observe: func(*rapid.T, int) int { return n },
			}
			if err := l.Check(rt, 0, 0); err == nil {
				rt.Fatal("a repeat call that changes observable state is not idempotent")
			}
		})
	})

	// Cycles and Tolerance both fall back to runtime-tuned defaults, so a law
	// left unconfigured still exercises the subject rather than doing nothing.
	t.Run("LeakFree defaults its cycle count and tolerance", func(t *testing.T) {
		t.Parallel()
		opens := 0
		l := law.LeakFree[int]{
			Open:  func(*rapid.T, int) error { opens++; return nil },
			Close: func(*rapid.T, int) error { return nil },
		}
		rapid.Check(t, func(rt *rapid.T) {
			opens = 0
			// The verdict is deliberately ignored: LeakFree samples
			// runtime.NumGoroutine() across the cycle, so a parallel suite can
			// move the count under it. What this covers is the defaulting —
			// an unset Cycles must still drive the lifecycle repeatedly rather
			// than running zero iterations and passing vacuously.
			_ = l.Check(rt, 0, 0)
			if opens < 2 {
				rt.Fatalf("the default cycle count must run several cycles, got %d", opens)
			}
		})
	})

	t.Run("LeakFree holds vacuously when Open or Close is refused", func(t *testing.T) {
		t.Parallel()
		refuseOpen := law.LeakFree[int]{
			Open:   func(*rapid.T, int) error { return errors.New("no") },
			Close:  func(*rapid.T, int) error { return nil },
			Cycles: 2, Tolerance: 1,
		}
		refuseClose := law.LeakFree[int]{
			Open:   func(*rapid.T, int) error { return nil },
			Close:  func(*rapid.T, int) error { return errors.New("no") },
			Cycles: 2, Tolerance: 1,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := refuseOpen.Check(rt, 0, 0); !law.Holds(err) {
				rt.Fatalf("a refused Open is a precondition: %v", err)
			}
			if err := refuseClose.Check(rt, 0, 0); !law.Holds(err) {
				rt.Fatalf("a refused Close is a precondition: %v", err)
			}
		})
	})

	t.Run("LifecycleAfterCloseSentinel holds vacuously when Close is refused", func(t *testing.T) {
		t.Parallel()
		sentinel := errors.New("closed")
		l := law.LifecycleAfterCloseSentinel[int]{
			Close:    func(*rapid.T, int) error { return errors.New("cannot close") },
			Op:       func(*rapid.T, int) error { return sentinel },
			Sentinel: sentinel,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); !law.Holds(err) {
				rt.Fatalf("a refused Close is a precondition: %v", err)
			}
		})
	})

	t.Run("LifecycleAfterCloseSentinel flags a wrong error after close", func(t *testing.T) {
		t.Parallel()
		sentinel := errors.New("closed")
		l := law.LifecycleAfterCloseSentinel[int]{
			Close:    func(*rapid.T, int) error { return nil },
			Op:       func(*rapid.T, int) error { return errors.New("something else") },
			Sentinel: sentinel,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); err == nil {
				rt.Fatal("an op returning the wrong error after close is a violation")
			}
		})
	})

	t.Run("LifecycleAfterCloseSentinel accepts a wrapped sentinel", func(t *testing.T) {
		t.Parallel()
		sentinel := errors.New("closed")
		l := law.LifecycleAfterCloseSentinel[int]{
			Close:    func(*rapid.T, int) error { return nil },
			Op:       func(*rapid.T, int) error { return fmt.Errorf("op: %w", sentinel) },
			Sentinel: sentinel,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); err != nil {
				rt.Fatalf("errors.Is must see through wrapping: %v", err)
			}
		})
	})

	// Poison that does not take is not a violation — there is nothing to stay
	// consistent about — but poison that heals on its own is.
	t.Run("PoisonConsistent skips when poisoning is a no-op", func(t *testing.T) {
		t.Parallel()
		l := law.PoisonConsistent[int]{
			Poison: func(*rapid.T, int) {},
			Probe:  func(*rapid.T, int) error { return nil },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); err != nil {
				rt.Fatalf("ineffective poisoning must skip, not fail: %v", err)
			}
		})
	})

	t.Run("PoisonConsistent flags poison that heals", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			probes := 0
			l := law.PoisonConsistent[int]{
				Poison: func(*rapid.T, int) { probes = 0 },
				Probe: func(*rapid.T, int) error {
					probes++
					if probes == 1 {
						return errors.New("poisoned")
					}
					return nil // heals on the next probe
				},
			}
			if err := l.Check(rt, 0, 0); err == nil {
				rt.Fatal("poison that clears itself is a violation")
			}
		})
	})

	t.Run("PoisonConsistent passes when poison persists", func(t *testing.T) {
		t.Parallel()
		l := law.PoisonConsistent[int]{
			Poison: func(*rapid.T, int) {},
			Probe:  func(*rapid.T, int) error { return errors.New("poisoned") },
			Reads:  0, // exercises the default probe count
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); err != nil {
				rt.Fatalf("persistent poison must pass: %v", err)
			}
		})
	})

	// LeakFree's verdict is the one thing the mechanism test above cannot
	// assert, because it samples the whole process. Leaking far more than the
	// tolerance makes the drift unambiguous regardless of what else is running.
	//nolint:paralleltest // goroutine counting needs exclusive control
	t.Run("LeakFree flags a cycle that leaks goroutines", func(t *testing.T) {
		const leaks = 256
		park := make(chan struct{})
		var spawned sync.WaitGroup
		l := law.LeakFree[*lifecycleSUT]{
			Open: func(*rapid.T, *lifecycleSUT) error {
				spawned.Go(func() { <-park }) // never returns until released
				return nil
			},
			Close:     func(*rapid.T, *lifecycleSUT) error { return nil },
			Cycles:    leaks,
			Tolerance: 1,
		}
		err := l.Check(nil, &lifecycleSUT{}, &lifecycleSUT{})
		close(park)
		spawned.Wait()

		if err == nil {
			t.Fatalf("%d parked goroutines must exceed a tolerance of 1", leaks)
		}
		if !strings.Contains(err.Error(), "goroutine count grew") {
			t.Fatalf("the diagnostic must name the drift, got: %v", err)
		}
	})
}
