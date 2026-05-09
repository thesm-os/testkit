// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package interfaces

import (
	"context"
	"io"
	"iter"
	"sync"
)

// InMemoryAllShapes is the [AllShapes] companion. Map-backed for
// reader/writer methods, slice-backed for streams, with explicit
// counters for lifecycle / aggregator / predicate methods so tests
// can observe state changes through DelegateTo.
//
// Goroutine-safe via a single sync.RWMutex. Reads grab RLock,
// mutations grab Lock.
type InMemoryAllShapes struct {
	mu sync.RWMutex

	// items is the read/write storage backing the reader, writer,
	// deleter, and stream methods.
	items map[string]Record

	// touches counts Touch invocations — exposes Mutator state for
	// DelegateTo verification.
	touches map[string]int

	// initCount / resetCount expose Lifecycle / VoidLifecycle state.
	initCount  int
	resetCount int

	// poison is the value Err() returns. Tests configure via
	// SetPoison to assert PoisonAccessor dispatch.
	poison error

	// healthy is the value IsHealthy() returns.
	healthy bool

	// description is the value Description() returns.
	description string
}

// NewInMemoryAllShapes returns an empty in-memory companion.
//
// The returned impl is non-poisoned, healthy=true, with
// description="in-memory". Tests that need different defaults
// override via the setters.
func NewInMemoryAllShapes() *InMemoryAllShapes {
	return &InMemoryAllShapes{
		items:       make(map[string]Record),
		touches:     make(map[string]int),
		healthy:     true,
		description: "in-memory",
	}
}

// Seed prepopulates the items map. Tests use it to set up state
// before wrapping the impl in a stub via DelegateTo.
func (s *InMemoryAllShapes) Seed(items ...Record) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, it := range items {
		s.items[it.ID] = it
	}
}

// SetPoison configures Err()'s return value.
func (s *InMemoryAllShapes) SetPoison(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.poison = err
}

// SetHealthy configures IsHealthy()'s return value.
func (s *InMemoryAllShapes) SetHealthy(healthy bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.healthy = healthy
}

// InitCount returns how many times Init has been called — used by
// tests asserting Lifecycle dispatch through DelegateTo.
func (s *InMemoryAllShapes) InitCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.initCount
}

// ResetCount returns how many times Reset has been called.
func (s *InMemoryAllShapes) ResetCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.resetCount
}

// Touches returns the touch counter for the named key.
func (s *InMemoryAllShapes) Touches(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.touches[key]
}

// All implements [AllShapes]. (StreamReader, iter.Seq.)
func (s *InMemoryAllShapes) All(_ context.Context) iter.Seq[Record] {
	return func(yield func(Record) bool) {
		s.mu.RLock()
		snapshot := make([]Record, 0, len(s.items))
		for _, v := range s.items {
			snapshot = append(snapshot, v)
		}
		s.mu.RUnlock()
		for _, v := range snapshot {
			if !yield(v) {
				return
			}
		}
	}
}

// Scan implements [AllShapes]. (StreamReader, iter.Seq2 with error.)
func (s *InMemoryAllShapes) Scan(_ context.Context) iter.Seq2[Record, error] {
	return func(yield func(Record, error) bool) {
		s.mu.RLock()
		snapshot := make([]Record, 0, len(s.items))
		for _, v := range s.items {
			snapshot = append(snapshot, v)
		}
		s.mu.RUnlock()
		for _, v := range snapshot {
			if !yield(v, nil) {
				return
			}
		}
	}
}

// Many implements [AllShapes]. (BatchReader.) Returns the records
// present for the given keys; missing keys are silently skipped.
func (s *InMemoryAllShapes) Many(_ context.Context, keys ...string) ([]Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Record, 0, len(keys))
	for _, k := range keys {
		if v, ok := s.items[k]; ok {
			out = append(out, v)
		}
	}
	return out, nil
}

// ReadFrom implements [AllShapes]. (StreamConsumer.) Reads to
// completion; returns the byte count.
func (s *InMemoryAllShapes) ReadFrom(_ context.Context, r io.Reader) (int, error) {
	if r == nil {
		return 0, nil
	}
	buf := make([]byte, 64)
	n, err := io.ReadFull(r, buf)
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		err = nil
	}
	return n, err
}

// Inspect implements [AllShapes]. (Lookup, 3-result with bool.)
func (s *InMemoryAllShapes) Inspect(_ context.Context, key string) (Record, string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[key]
	if !ok {
		return Record{}, "", false
	}
	return v, "meta:" + v.ID, true
}

// Load implements [AllShapes]. (ReaderWithBool, comma-ok.)
func (s *InMemoryAllShapes) Load(_ context.Context, key string) (Record, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[key]
	return v, ok
}

// Err implements [AllShapes]. (PoisonAccessor.) Returns the
// configured poison or nil.
func (s *InMemoryAllShapes) Err() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.poison
}

// IsHealthy implements [AllShapes]. (Predicate.)
func (s *InMemoryAllShapes) IsHealthy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.healthy
}

// Reset implements [AllShapes]. (VoidLifecycle.)
func (s *InMemoryAllShapes) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	clear(s.items)
	clear(s.touches)
	s.poison = nil
	s.resetCount++
}

// Description implements [AllShapes]. (Pure.)
func (s *InMemoryAllShapes) Description() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.description
}

// Schedule implements [AllShapes]. (MultiArgWriter.) Stores the
// value under the key; priority is recorded as a metadata note in
// the value's Value field for tests that want to observe it.
func (s *InMemoryAllShapes) Schedule(_ context.Context, key string, value Record, priority int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = value
	_ = priority
	return nil
}

// Set implements [AllShapes]. (CompositeWriter.)
func (s *InMemoryAllShapes) Set(_ context.Context, key string, value Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = value
	return nil
}

// Fetch implements [AllShapes]. (MultiReader.)
func (s *InMemoryAllShapes) Fetch(_ context.Context, key string) (Record, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[key]
	if !ok {
		return Record{}, "", ErrNotFound
	}
	return v, "meta:" + v.ID, nil
}

// Stats implements [AllShapes]. (MultiAggregator.) Returns
// (item count, touch sum, nil).
func (s *InMemoryAllShapes) Stats(_ context.Context) (int, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	totalTouches := 0
	for _, n := range s.touches {
		totalTouches += n
	}
	return len(s.items), totalTouches, nil
}

// Remove implements [AllShapes]. (Deleter.)
func (s *InMemoryAllShapes) Remove(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, key)
	return nil
}

// Put implements [AllShapes]. (Writer.) Keys by item.ID.
func (s *InMemoryAllShapes) Put(_ context.Context, item Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[item.ID] = item
	return nil
}

// Find implements [AllShapes]. (PointerReader.)
func (s *InMemoryAllShapes) Find(_ context.Context, key string) *Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[key]
	if !ok {
		return nil
	}
	return &v
}

// Get implements [AllShapes]. (Reader.) Returns ErrNotFound on miss.
func (s *InMemoryAllShapes) Get(_ context.Context, key string) (Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[key]
	if !ok {
		return Record{}, ErrNotFound
	}
	return v, nil
}

// Lookup implements [AllShapes]. (ReaderNoError.) Returns the zero
// Record on miss.
func (s *InMemoryAllShapes) Lookup(_ context.Context, key string) Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.items[key]
}

// Count implements [AllShapes]. (Aggregator.)
func (s *InMemoryAllShapes) Count(_ context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items), nil
}

// Touch implements [AllShapes]. (Mutator.)
func (s *InMemoryAllShapes) Touch(_ context.Context, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.touches[key]++
}

// Init implements [AllShapes]. (Lifecycle.)
func (s *InMemoryAllShapes) Init(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.initCount++
	return nil
}

// Statistics implements [AllShapes]. (Unknown — 3 non-error
// returns falls through MultiAggregator's 2-result cap.)
func (s *InMemoryAllShapes) Statistics(_ context.Context) (int, int, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	totalTouches := 0
	for _, n := range s.touches {
		totalTouches += n
	}
	return len(s.items), totalTouches, s.resetCount, nil
}
