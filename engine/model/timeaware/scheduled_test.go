// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package timeaware_test

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model/law"
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
		// Two fixtures under one advance — the deployed shape: every
		// accepted schedule mirrors onto the reference, and the shared
		// clock fires both together.
		sut := newSched(time.Unix(0, 0))
		ref := newSched(time.Unix(0, 0))
		l := timeaware.ScheduledFiresAfterAdvance[*schedFixture]{
			Schedule: func(_ *rapid.T, s *schedFixture, at time.Duration) error {
				s.schedule(at)
				return nil
			},
			FiredCount: func(_ *rapid.T, s *schedFixture) int { return s.firedCount() },
			Offsets:    rapid.SampledFrom([]time.Duration{time.Second, 2 * time.Second, 3 * time.Second}),
			N:          3,
			Advance: func(d time.Duration) {
				sut.advance(d)
				ref.advance(d)
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, sut, ref); err != nil {
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
		ref := newSched(time.Unix(0, 0))
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, ref); err == nil {
				rt.Fatal("expected missing-fires flagged")
			}
		})
	})
}

// The scheduling law tolerates a subject that refuses some schedules — those
// are preconditions — but a subject that accepts none leaves nothing to
// verify, and one that accepts and then never fires is the real defect.
func TestScheduledFiresAfterAdvanceBranches(t *testing.T) {
	t.Parallel()

	type sched struct {
		accepted  int
		fired     int
		refuse    bool
		neverFire bool
	}
	mk := func(s *sched, n int) timeaware.ScheduledFiresAfterAdvance[*sched] {
		return timeaware.ScheduledFiresAfterAdvance[*sched]{
			Schedule: func(_ *rapid.T, x *sched, _ time.Duration) error {
				if x.refuse {
					return errors.New("queue full")
				}
				x.accepted++
				return nil
			},
			FiredCount: func(_ *rapid.T, x *sched) int { return x.fired },
			Offsets:    rapid.Just(time.Second),
			N:          n,
			Advance: func(time.Duration) {
				if !s.neverFire {
					s.fired = s.accepted
				}
			},
		}
	}

	t.Run("every accepted task fires after the advance", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			// Distinct pair: every accepted schedule mirrors onto the
			// reference, and a shared instance would double-count itself.
			s := &sched{}
			if err := mk(s, 3).Check(rt, s, &sched{}); err != nil {
				rt.Fatalf("all scheduled work fired: %v", err)
			}
		})
	})

	t.Run("a subject that refuses everything holds vacuously", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			s := &sched{refuse: true}
			if err := mk(s, 3).Check(rt, s, &sched{}); !law.Holds(err) {
				rt.Fatalf("nothing scheduled means nothing to verify: %v", err)
			}
		})
	})

	t.Run("scheduled work that never fires is a violation", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			s := &sched{neverFire: true}
			if err := mk(s, 3).Check(rt, s, &sched{}); err == nil {
				rt.Fatal("work that never fires after the advance is a violation")
			}
		})
	})

	t.Run("a reference that refuses a mirrored schedule is the divergence", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			s := &sched{}
			err := mk(s, 3).Check(rt, s, &sched{refuse: true})
			if err == nil || !strings.Contains(err.Error(), "the reference refused") {
				rt.Fatalf("a refusing reference is the law's own finding, got: %v", err)
			}
		})
	})

	// An unset N schedules a default batch rather than zero tasks, which would
	// make the law vacuous for every subject.
	t.Run("a non-positive N defaults the batch size", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			s := &sched{}
			if err := mk(s, 0).Check(rt, s, &sched{}); err != nil {
				rt.Fatalf("the default batch must still verify: %v", err)
			}
			if s.accepted < 2 {
				rt.Fatalf("the default batch must schedule several tasks, got %d", s.accepted)
			}
		})
	})

	t.Run("identity", func(t *testing.T) {
		t.Parallel()
		var l timeaware.ScheduledFiresAfterAdvance[*sched]
		if l.ID() != "AUTO-SCHEDULED-FIRES-AFTER-ADVANCE" {
			t.Fatalf("unexpected law ID %q", l.ID())
		}
		if l.REQID() != "" {
			t.Fatalf("auto-derived laws carry no REQ tag, got %q", l.REQID())
		}
	})
}
