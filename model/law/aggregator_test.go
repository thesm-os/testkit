// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/law"
	"go.thesmos.sh/testkit/model/refwindow"
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
		l := law.Windowed[*refwindow.RollingCounter[string], string]{
			Incr: func(rt *rapid.T, c *refwindow.RollingCounter[string], k string) error { return c.Incr(rt.Context(), k) },
			Count: func(rt *rapid.T, c *refwindow.RollingCounter[string], k string) (int, error) {
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
			c := refwindow.NewRollingCounter[string](window, now)
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
