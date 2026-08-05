// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package timeaware_test

import (
	"errors"
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
