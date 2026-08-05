// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model/law"
	"go.thesmos.sh/testkit/engine/model/ref"
)

type aggSUT struct {
	count int
	sum   int64
}

func TestAggregatorBounded(t *testing.T) {
	t.Parallel()

	t.Run("value inside range passes", func(t *testing.T) {
		t.Parallel()
		s := &aggSUT{count: 5}
		l := law.AggregatorBounded[*aggSUT, int]{
			Read: func(_ *rapid.T, a *aggSUT) (int, error) { return a.count, nil },
			Min:  0, Max: 10,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("value above max flagged", func(t *testing.T) {
		t.Parallel()
		s := &aggSUT{count: 99}
		l := law.AggregatorBounded[*aggSUT, int]{
			Read: func(_ *rapid.T, a *aggSUT) (int, error) { return a.count, nil },
			Min:  0, Max: 10,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected out-of-range")
			}
		})
	})

	t.Run("value below min flagged", func(t *testing.T) {
		t.Parallel()
		s := &aggSUT{count: -1}
		l := law.AggregatorBounded[*aggSUT, int]{
			Read: func(_ *rapid.T, a *aggSUT) (int, error) { return a.count, nil },
			Min:  0, Max: 10,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected out-of-range")
			}
		})
	})

	t.Run("error skips check (vacuous)", func(t *testing.T) {
		t.Parallel()
		l := law.AggregatorBounded[*aggSUT, int]{
			Read: func(_ *rapid.T, _ *aggSUT) (int, error) { return 0, errors.New("nope") },
			Min:  0, Max: 10,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &aggSUT{}, &aggSUT{}); err != nil {
				rt.Fatal(err)
			}
		})
	})
}

func TestAssociative(t *testing.T) {
	t.Parallel()

	t.Run("integer addition is associative", func(t *testing.T) {
		t.Parallel()
		l := law.Associative[*aggSUT, int, int64]{
			Factory: func() *aggSUT { return &aggSUT{} },
			Apply:   func(_ *rapid.T, a *aggSUT, v int) error { a.sum += int64(v); return nil },
			Values:  rapid.IntRange(0, 100),
			Observe: func(_ *rapid.T, a *aggSUT) int64 { return a.sum },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &aggSUT{}, &aggSUT{}); err != nil {
				rt.Fatal(err)
			}
		})
	})
}

func TestConservative(t *testing.T) {
	t.Parallel()

	t.Run("preserved sum across writes passes", func(t *testing.T) {
		t.Parallel()
		s := &aggSUT{sum: 100}
		l := law.Conservative[*aggSUT, int]{
			Sum:    func(_ *rapid.T, a *aggSUT) int64 { return a.sum },
			Write:  func(_ *rapid.T, _ *aggSUT, _ int) error { return nil }, // no-op preserves sum
			Values: rapid.IntRange(0, 10),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("write that changes sum is flagged", func(t *testing.T) {
		t.Parallel()
		s := &aggSUT{sum: 100}
		l := law.Conservative[*aggSUT, int]{
			Sum:    func(_ *rapid.T, a *aggSUT) int64 { return a.sum },
			Write:  func(_ *rapid.T, a *aggSUT, v int) error { a.sum += int64(v); return nil },
			Values: rapid.IntRange(1, 10), // strictly positive → sum drifts
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected sum drift")
			}
		})
	})
}

// stickyCounter is a broken windowed counter: it counts increments
// but never decays them, ignoring the window entirely.
type stickyCounter struct {
	n map[string]int
}

func (c *stickyCounter) incr(k string) error {
	if c.n == nil {
		c.n = make(map[string]int)
	}
	c.n[k]++
	return nil
}

func (c *stickyCounter) count(k string) (int, error) { return c.n[k], nil }

func TestWindowed(t *testing.T) {
	t.Parallel()

	window := 10 * time.Second

	t.Run("rolling counter decays increments past the window", func(t *testing.T) {
		t.Parallel()
		l := law.Windowed[*ref.RollingCounter[string], string]{
			Incr: func(rt *rapid.T, c *ref.RollingCounter[string], k string) error { return c.Incr(rt.Context(), k) },
			Count: func(rt *rapid.T, c *ref.RollingCounter[string], k string) (int, error) {
				return c.Count(rt.Context(), k)
			},
			Keys:   rapid.SampledFrom([]string{"a", "b"}),
			Window: window,
		}
		rapid.Check(t, func(rt *rapid.T) {
			var mu sync.Mutex
			nowT := time.Unix(0, 0)
			now := func() time.Time { mu.Lock(); defer mu.Unlock(); return nowT }
			l.Advance = func(d time.Duration) { mu.Lock(); nowT = nowT.Add(d); mu.Unlock() }
			c := ref.NewRollingCounter[string](window, now)
			if err := l.Check(rt, c, c); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("counter that never decays is caught", func(t *testing.T) {
		t.Parallel()
		l := law.Windowed[*stickyCounter, string]{
			Incr:    func(_ *rapid.T, c *stickyCounter, k string) error { return c.incr(k) },
			Count:   func(_ *rapid.T, c *stickyCounter, k string) (int, error) { return c.count(k) },
			Advance: func(time.Duration) {}, // sticky counter has no clock
			Keys:    rapid.SampledFrom([]string{"a", "b"}),
			Window:  window,
		}
		rapid.Check(t, func(rt *rapid.T) {
			c := &stickyCounter{}
			if err := l.Check(rt, c, c); err == nil {
				rt.Fatal("expected non-decaying counter to be caught")
			}
		})
	})
}

// The aggregator laws share a shape: build state through Apply/Write, observe
// it, and compare. A subject that refuses the setup has failed a precondition;
// one that accepts it and then disagrees has violated the law.
func TestAggregatorLawBranches(t *testing.T) {
	t.Parallel()

	t.Run("Associative holds vacuously when an Apply is refused", func(t *testing.T) {
		t.Parallel()
		// Every Apply fails, so neither side ever gets built.
		l := law.Associative[*intBag, int, int]{
			Factory: func() *intBag { return &intBag{} },
			Apply:   func(*rapid.T, *intBag, int) error { return errors.New("closed") },
			Observe: func(_ *rapid.T, b *intBag) int { return b.sum },
			Values:  rapid.Just(1),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); err != nil {
				rt.Fatalf("a refused Apply is a precondition: %v", err)
			}
		})
	})

	// Order-sensitive aggregation is the defect: (a;b);c and a;(b;c) must agree
	// observationally, so an implementation that records arrival order fails.
	t.Run("Associative flags order-sensitive aggregation", func(t *testing.T) {
		t.Parallel()
		calls := 0
		l := law.Associative[*intBag, int, int]{
			Factory: func() *intBag { return &intBag{} },
			Apply: func(_ *rapid.T, b *intBag, v int) error {
				calls++
				b.sum = b.sum*10 + v + calls // deliberately path-dependent
				return nil
			},
			Observe: func(_ *rapid.T, b *intBag) int { return b.sum },
			Values:  rapid.Just(1),
		}
		rapid.Check(t, func(rt *rapid.T) {
			calls = 0
			if err := l.Check(rt, nil, nil); err == nil {
				rt.Fatal("a path-dependent aggregate is not associative")
			}
		})
	})

	t.Run("Conservative holds vacuously when the write is refused", func(t *testing.T) {
		t.Parallel()
		l := law.Conservative[*intBag, int]{
			Sum:    func(_ *rapid.T, b *intBag) int64 { return int64(b.sum) },
			Write:  func(*rapid.T, *intBag, int) error { return errors.New("read-only") },
			Values: rapid.Just(1),
		}
		rapid.Check(t, func(rt *rapid.T) {
			b := &intBag{}
			if err := l.Check(rt, b, b); err != nil {
				rt.Fatalf("a refused write is a precondition: %v", err)
			}
		})
	})

	t.Run("Conservative flags a write that changes the total", func(t *testing.T) {
		t.Parallel()
		l := law.Conservative[*intBag, int]{
			Sum:    func(_ *rapid.T, b *intBag) int64 { return int64(b.sum) },
			Write:  func(_ *rapid.T, b *intBag, v int) error { b.sum += v; return nil },
			Values: rapid.Just(1),
		}
		rapid.Check(t, func(rt *rapid.T) {
			b := &intBag{}
			if err := l.Check(rt, b, b); err == nil {
				rt.Fatal("a write that moves the conserved total is a violation")
			}
		})
	})

	t.Run("Conservative passes when the total is preserved", func(t *testing.T) {
		t.Parallel()
		l := law.Conservative[*intBag, int]{
			Sum:    func(_ *rapid.T, b *intBag) int64 { return int64(b.sum) },
			Write:  func(*rapid.T, *intBag, int) error { return nil }, // moves nothing
			Values: rapid.Just(1),
		}
		rapid.Check(t, func(rt *rapid.T) {
			b := &intBag{}
			if err := l.Check(rt, b, b); err != nil {
				rt.Fatalf("a total-preserving write must pass: %v", err)
			}
		})
	})
}

// A windowed counter must both reflect an increment and let it decay once the
// clock passes the window. Failing either half is a distinct defect, and a
// counter that errors mid-sequence is a precondition rather than a violation.
func TestWindowedLawBranches(t *testing.T) {
	t.Parallel()

	type counter struct {
		n       int
		decayed bool
		err     error
	}
	mk := func(c *counter, decay bool) law.Windowed[*counter, string] {
		return law.Windowed[*counter, string]{
			Count: func(_ *rapid.T, s *counter, _ string) (int, error) {
				if s.err != nil {
					return 0, s.err
				}
				if decay && s.decayed {
					return 0, nil
				}
				return s.n, nil
			},
			Incr:    func(_ *rapid.T, s *counter, _ string) error { s.n++; return nil },
			Advance: func(time.Duration) { c.decayed = true },
			Keys:    rapid.Just("k"),
			Window:  time.Second,
		}
	}

	t.Run("a decaying counter passes", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			c := &counter{}
			l := mk(c, true)
			if err := l.Check(rt, c, c); err != nil {
				rt.Fatalf("a counter that decays past the window must pass: %v", err)
			}
		})
	})

	t.Run("a counter that never decays is a violation", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			c := &counter{}
			l := mk(c, false) // ignores the window
			if err := l.Check(rt, c, c); err == nil {
				rt.Fatal("a count that survives the window is a violation")
			}
		})
	})

	t.Run("a failing initial count holds vacuously", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			c := &counter{err: errors.New("unavailable")}
			l := mk(c, true)
			if err := l.Check(rt, c, c); err != nil {
				rt.Fatalf("an unreadable counter is a precondition: %v", err)
			}
		})
	})

	t.Run("a refused increment holds vacuously", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			c := &counter{}
			l := mk(c, true)
			l.Incr = func(*rapid.T, *counter, string) error { return errors.New("throttled") }
			if err := l.Check(rt, c, c); err != nil {
				rt.Fatalf("a refused increment is a precondition: %v", err)
			}
		})
	})

	t.Run("an increment not reflected in the count is a violation", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			c := &counter{}
			l := mk(c, true)
			l.Incr = func(*rapid.T, *counter, string) error { return nil } // silently drops
			if err := l.Check(rt, c, c); err == nil {
				rt.Fatal("an increment that does not raise the count is a violation")
			}
		})
	})
}

// intBag is a minimal accumulator for the aggregator laws. refuse makes it
// decline input, which is how the laws' precondition arms are reached without
// standing up a real aggregator.
type intBag struct {
	sum    int
	refuse bool
}

func TestAggregatorLawPreconditionsAndViolations(t *testing.T) {
	t.Parallel()

	boom := errors.New("refused")

	// Associative builds two independent groupings; a fold that refuses
	// input on the second one is just as much a precondition as on the first.
	t.Run("Associative holds vacuously when the second grouping is refused", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			groupings := 0
			l := law.Associative[*intBag, int, int]{
				Factory: func() *intBag {
					groupings++
					return &intBag{refuse: groupings > 1}
				},
				Apply: func(_ *rapid.T, b *intBag, v int) error {
					if b.refuse {
						return boom
					}
					b.sum += v
					return nil
				},
				Values:  rapid.IntRange(0, 5),
				Observe: func(_ *rapid.T, b *intBag) int { return b.sum },
			}
			if err := l.Check(rt, nil, nil); err != nil {
				rt.Fatalf("a refused second grouping is a precondition: %v", err)
			}
		})
	})

	// Once the increment lands, a Count that errors is the subject
	// contradicting itself — it answered a moment ago.
	t.Run("Windowed flags a Count that fails after the increment", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			counts := 0
			l := law.Windowed[*intBag, string]{
				Incr: func(*rapid.T, *intBag, string) error { return nil },
				Count: func(*rapid.T, *intBag, string) (int, error) {
					counts++
					if counts > 1 {
						return 0, boom
					}
					return 0, nil
				},
				Advance: func(time.Duration) {},
				Keys:    rapid.Just("k"),
				Window:  time.Second,
			}
			err := l.Check(rt, &intBag{}, &intBag{})
			if err == nil || !strings.Contains(err.Error(), "count after increment errored") {
				rt.Fatalf("a Count that stops answering must be reported, got: %v", err)
			}
		})
	})

	t.Run("Windowed flags a Count that fails after the window advances", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			counts := 0
			l := law.Windowed[*intBag, string]{
				Incr: func(*rapid.T, *intBag, string) error { return nil },
				Count: func(*rapid.T, *intBag, string) (int, error) {
					counts++
					if counts > 2 {
						return 0, boom
					}
					return counts, nil
				},
				Advance: func(time.Duration) {},
				Keys:    rapid.Just("k"),
				Window:  time.Second,
			}
			err := l.Check(rt, &intBag{}, &intBag{})
			if err == nil || !strings.Contains(err.Error(), "count after advance errored") {
				rt.Fatalf("a Count that stops answering must be reported, got: %v", err)
			}
		})
	})
}
