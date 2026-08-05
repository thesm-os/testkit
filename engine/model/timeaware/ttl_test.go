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

	"go.thesmos.sh/testkit/engine/model/timeaware"
)

// ttlStore is a hand-rolled TTL fixture used to drive the TTL law.
type ttlStore struct {
	mu      sync.Mutex
	now     time.Time
	entries map[string]ttlEntry
}

type ttlEntry struct {
	value     string
	expiresAt time.Time
}

func newTTLStore(now time.Time) *ttlStore {
	return &ttlStore{now: now, entries: make(map[string]ttlEntry)}
}

func (s *ttlStore) put(k, v string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[k] = ttlEntry{value: v, expiresAt: s.now.Add(ttl)}
}

func (s *ttlStore) get(k string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[k]
	if !ok || !e.expiresAt.After(s.now) {
		return "", false
	}
	return e.value, true
}

func (s *ttlStore) advance(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.now = s.now.Add(d)
}

var errTTLNotFound = errors.New("ttl: not found")

func TestTTLExpiryAfterAdvance(t *testing.T) {
	t.Parallel()

	t.Run("compliant TTL store passes", func(t *testing.T) {
		t.Parallel()
		s := newTTLStore(time.Unix(0, 0))
		ttl := 10 * time.Second
		l := timeaware.TTLExpiryAfterAdvance[*ttlStore, string, string]{
			Put: func(_ *rapid.T, s *ttlStore, k, v string) error {
				s.put(k, v, ttl)
				return nil
			},
			Read: func(_ *rapid.T, s *ttlStore, k string) (string, error) {
				v, ok := s.get(k)
				if !ok {
					return "", errTTLNotFound
				}
				return v, nil
			},
			Keys:     rapid.SampledFrom([]string{"k1"}),
			Values:   rapid.SampledFrom([]string{"v1", "v2"}),
			TTL:      ttl,
			Advance:  s.advance,
			NotFound: errTTLNotFound,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("non-expiring store flagged", func(t *testing.T) {
		t.Parallel()
		s := newTTLStore(time.Unix(0, 0))
		ttl := 10 * time.Second
		// Make read return the stored value forever (ignoring expiry).
		l := timeaware.TTLExpiryAfterAdvance[*ttlStore, string, string]{
			Put: func(_ *rapid.T, s *ttlStore, k, v string) error {
				s.put(k, v, ttl)
				return nil
			},
			Read: func(_ *rapid.T, s *ttlStore, k string) (string, error) {
				s.mu.Lock()
				defer s.mu.Unlock()
				e, ok := s.entries[k]
				if !ok {
					return "", errTTLNotFound
				}
				return e.value, nil // ignores expiry
			},
			Keys:     rapid.SampledFrom([]string{"k1"}),
			Values:   rapid.SampledFrom([]string{"v1"}),
			TTL:      ttl,
			Advance:  s.advance,
			NotFound: errTTLNotFound,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected expiry violation")
			}
		})
	})
}

// TTL expiry has three outcomes the law must keep apart: a refused write is a
// precondition, an entry unreadable *before* the advance means the subject
// never stored it, and an entry still readable *after* the advance is the
// expiry failure.
func TestTTLExpiryAfterAdvanceBranches(t *testing.T) {
	t.Parallel()

	notFound := errors.New("not found")
	type store struct {
		data    map[string]string
		putErr  error
		expired bool
		noStore bool
	}
	mk := func(s *store, ttl time.Duration) timeaware.TTLExpiryAfterAdvance[*store, string, string] {
		return timeaware.TTLExpiryAfterAdvance[*store, string, string]{
			Put: func(_ *rapid.T, x *store, k, v string) error {
				if x.putErr != nil {
					return x.putErr
				}
				if !x.noStore {
					x.data[k] = v
				}
				return nil
			},
			Read: func(_ *rapid.T, x *store, k string) (string, error) {
				if x.expired {
					return "", notFound
				}
				v, ok := x.data[k]
				if !ok {
					return "", notFound
				}
				return v, nil
			},
			Keys:     rapid.Just("k"),
			Values:   rapid.Just("v"),
			TTL:      ttl,
			NotFound: notFound,
			Advance:  func(time.Duration) { s.expired = true },
		}
	}
	fresh := func() *store { return &store{data: map[string]string{}} }

	t.Run("an entry that expires passes", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			s := fresh()
			if err := mk(s, time.Second).Check(rt, s, s); err != nil {
				rt.Fatalf("an entry that expires on schedule must pass: %v", err)
			}
		})
	})

	t.Run("a refused Put holds vacuously", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			s := fresh()
			s.putErr = errors.New("read-only")
			if err := mk(s, time.Second).Check(rt, s, s); err != nil {
				rt.Fatalf("a refused write is a precondition: %v", err)
			}
		})
	})

	// A store that accepted the write but cannot read it back has a bug that
	// has nothing to do with TTL, and the law says so rather than blaming
	// expiry.
	t.Run("an entry unreadable before the advance is a violation", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			s := fresh()
			s.noStore = true
			err := mk(s, time.Second).Check(rt, s, s)
			if err == nil {
				rt.Fatal("a stored entry must be readable before its TTL elapses")
			}
			if !strings.Contains(err.Error(), "pre-advance") {
				rt.Fatalf("the diagnostic must locate the failure, got: %v", err)
			}
		})
	})

	t.Run("an entry that outlives its TTL is a violation", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			s := fresh()
			l := mk(s, time.Second)
			l.Advance = func(time.Duration) {} // clock moves, entry does not expire
			err := l.Check(rt, s, s)
			if err == nil {
				rt.Fatal("an entry readable past its TTL is a violation")
			}
			if !strings.Contains(err.Error(), "post-advance") {
				rt.Fatalf("the diagnostic must locate the failure, got: %v", err)
			}
		})
	})

	t.Run("identity", func(t *testing.T) {
		t.Parallel()
		var l timeaware.TTLExpiryAfterAdvance[*store, string, string]
		if l.ID() != "AUTO-TTL-EXPIRY" {
			t.Fatalf("unexpected law ID %q", l.ID())
		}
		if l.REQID() != "" {
			t.Fatalf("auto-derived laws carry no REQ tag, got %q", l.REQID())
		}
	})
}
