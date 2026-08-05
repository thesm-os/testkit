// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package timeaware_test

import (
	"sync"
	"testing"
	"time"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model/timeaware"
)

// schedFixture is a hand-rolled scheduler used to drive the
// scheduled-fires law.
type schedFixture struct {
	mu      sync.Mutex
	now     time.Time
	pending []time.Time
	fired   int
}

func newSched(now time.Time) *schedFixture {
	return &schedFixture{now: now}
}

func (s *schedFixture) schedule(at time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending = append(s.pending, s.now.Add(at))
}

func (s *schedFixture) firedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.fired
}

func (s *schedFixture) advance(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.now = s.now.Add(d)
	kept := s.pending[:0]
	for _, at := range s.pending {
		if !at.After(s.now) {
			s.fired++
			continue
		}
		kept = append(kept, at)
	}
	s.pending = kept
}

func TestScheduledFiresAfterAdvance(t *testing.T) {
	t.Parallel()

	t.Run("compliant scheduler fires every scheduled task", func(t *testing.T) {
		t.Parallel()
		s := newSched(time.Unix(0, 0))
		l := timeaware.ScheduledFiresAfterAdvance[*schedFixture]{
			Schedule: func(_ *rapid.T, s *schedFixture, at time.Duration) error {
				s.schedule(at)
				return nil
			},
			FiredCount: func(_ *rapid.T, s *schedFixture) int { return s.firedCount() },
			Offsets:    rapid.SampledFrom([]time.Duration{time.Second, 2 * time.Second, 3 * time.Second}),
			N:          3,
			Advance:    s.advance,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("scheduler that misses fires is flagged", func(t *testing.T) {
		t.Parallel()
		s := newSched(time.Unix(0, 0))
		// Override advance to never fire.
		stuck := func(d time.Duration) {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.now = s.now.Add(d)
			// intentionally do nothing about pending
		}
		l := timeaware.ScheduledFiresAfterAdvance[*schedFixture]{
			Schedule:   func(_ *rapid.T, s *schedFixture, at time.Duration) error { s.schedule(at); return nil },
			FiredCount: func(_ *rapid.T, s *schedFixture) int { return s.firedCount() },
			Offsets:    rapid.Just(time.Second),
			N:          2,
			Advance:    stuck,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected missing-fires flagged")
			}
		})
	})
}
