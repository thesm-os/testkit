// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package timeaware

import (
	"context"
	"sync"
	"time"

	"go.thesmos.sh/testkit/clock"
)

type timedItem struct {
	item      Item
	expiresAt time.Time
}

// InMemoryStore implements [Store] with TTL-based expiry using an
// injected [clock.Clock]. Items expire after [DefaultTTL] from Put.
type InMemoryStore struct {
	mu    sync.Mutex
	clock clock.Clock
	data  map[string]timedItem
}

// NewInMemoryStore returns a Store backed by the given clock.
func NewInMemoryStore(c clock.Clock) *InMemoryStore {
	return &InMemoryStore{
		clock: c,
		data:  make(map[string]timedItem),
	}
}

func (s *InMemoryStore) Get(ctx context.Context, id string) (Item, error) {
	if err := ctx.Err(); err != nil {
		return Item{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ti, ok := s.data[id]
	if !ok || s.clock.Now().After(ti.expiresAt) {
		if ok {
			delete(s.data, id)
		}
		return Item{}, ErrNotFound
	}
	return ti.item, nil
}

func (s *InMemoryStore) Put(ctx context.Context, item Item) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[item.ID] = timedItem{
		item:      item,
		expiresAt: s.clock.Now().Add(DefaultTTL),
	}
	return nil
}

func (s *InMemoryStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
	return nil
}

func (s *InMemoryStore) Count(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.clock.Now()
	n := 0
	for _, ti := range s.data {
		if !now.After(ti.expiresAt) {
			n++
		}
	}
	return n, nil
}

// BrokenTTLStore ignores the injected clock for TTL — uses time.Now()
// directly. Under TestClock, time never advances past origin, so items
// never expire. The framework detects the divergence: ref (correct TTL
// via TestClock) expires items, but SUT (real time) does not.
type BrokenTTLStore struct {
	mu   sync.Mutex
	clk  clock.Clock // stored but only used for Put timestamp
	data map[string]timedItem
}

// NewBrokenTTLStore returns a Store that ignores the clock for expiry.
func NewBrokenTTLStore(c clock.Clock) *BrokenTTLStore {
	return &BrokenTTLStore{
		clk:  c,
		data: make(map[string]timedItem),
	}
}

func (s *BrokenTTLStore) Get(ctx context.Context, id string) (Item, error) {
	if err := ctx.Err(); err != nil {
		return Item{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ti, ok := s.data[id]
	if !ok {
		return Item{}, ErrNotFound
	}
	// BUG: uses real time instead of injected clock.
	// Under TestClock, time.Now() stays near the real wall clock
	// while the TestClock advances far past TTL. So this never
	// sees items as expired when the ref does.
	if time.Now().After(ti.expiresAt) {
		delete(s.data, id)
		return Item{}, ErrNotFound
	}
	return ti.item, nil
}

func (s *BrokenTTLStore) Put(ctx context.Context, item Item) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Uses the injected clock for the expiry timestamp —
	// same as the correct implementation.
	s.data[item.ID] = timedItem{
		item:      item,
		expiresAt: s.clk.Now().Add(DefaultTTL),
	}
	return nil
}

func (s *BrokenTTLStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
	return nil
}

func (s *BrokenTTLStore) Count(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// BUG: uses real time for expiry check.
	now := time.Now()
	n := 0
	for _, ti := range s.data {
		if !now.After(ti.expiresAt) {
			n++
		}
	}
	return n, nil
}
