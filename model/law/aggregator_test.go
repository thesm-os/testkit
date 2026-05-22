// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"errors"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/law"
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
