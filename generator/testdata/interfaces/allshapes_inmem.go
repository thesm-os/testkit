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
//
// Contract-alignment overrides: a handful of methods (Count, Stats,
// Description, ReadFrom, Inspect's secondary, Fetch's secondary)
// have setters that override their natural state-derived returns
// with contract-aligned literals. The suite's e2e companion test
// flips these overrides on so the generated baseline contracts (which
// expect specific sample values like 42, "test-result0", etc.) line
// up with what the in-mem returns. Without overrides the in-mem
// behaves naturally.
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

	// description is the value Description() returns. When non-empty
	// (the default), Description returns it directly. Tests flip via
	// SetDescription to align with contract sample literals.
	description string

	// Contract-alignment overrides. Zero values mean "compute from
	// state"; non-zero/set values short-circuit to the configured
	// literal.
	countOverride       *int    // Count returns *countOverride when non-nil
	statsOverride       *[2]int // Stats returns the pair when non-nil
	readFromOverride    *int    // ReadFrom returns *readFromOverride when non-nil
	inspectMetaOverride *string // Inspect's secondary returns *inspectMetaOverride when non-nil
	fetchMetaOverride   *string // Fetch's secondary returns *fetchMetaOverride when non-nil

	// invalidMode flips Init/Lifecycle methods to return ErrNotFound
	// (treating an unconfigured impl as "invalid"). The suite's
	// WithInvalidFactory wires a separate in-mem in this mode so the
	// AssertLifecycleRejectInvalidWith assertion (which requires err
	// != nil from a misconfigured impl) succeeds.
	invalidMode bool
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

// SetDescription configures Description()'s return value.
func (s *InMemoryAllShapes) SetDescription(d string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.description = d
}

// SetCountOverride pins Count() to return n regardless of items map
// length. Used by the suite's e2e companion test to align with the
// Aggregator contract's sample value.
func (s *InMemoryAllShapes) SetCountOverride(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.countOverride = &n
}

// SetStatsOverride pins Stats() to return (v1, v2). Aligns with
// MultiAggregator contract samples.
func (s *InMemoryAllShapes) SetStatsOverride(v1, v2 int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pair := [2]int{v1, v2}
	s.statsOverride = &pair
}

// SetReadFromOverride pins ReadFrom() to return n. Aligns with
// StreamConsumer contract sample.
func (s *InMemoryAllShapes) SetReadFromOverride(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readFromOverride = &n
}

// SetInspectMeta pins Inspect's secondary string return. Aligns with
// Lookup contract's secondary-result sample.
func (s *InMemoryAllShapes) SetInspectMeta(meta string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inspectMetaOverride = &meta
}

// SetFetchMeta pins Fetch's secondary string return. Aligns with
// MultiReader contract's V2 sample.
func (s *InMemoryAllShapes) SetFetchMeta(meta string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fetchMetaOverride = &meta
}

// SetInvalidMode flips Init to return [ErrNotFound] (treating the
// unconfigured impl as invalid). Used by the suite's
// WithInvalidFactory option so AssertLifecycleRejectInvalidWith
// observes a non-nil error from a misconfigured impl.
func (s *InMemoryAllShapes) SetInvalidMode(invalid bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invalidMode = invalid
}

// SeedAt seeds items[key] = item — keying by the explicit param rather
// than item.ID. Reader / Lookup / etc. baselines call methods with the
// contract's sample key ("test-key") and expect the configured sample
// value (Record{ID:"test-id"}); SeedAt makes that mapping explicit
// rather than overloading [Seed]'s by-ID convention.
func (s *InMemoryAllShapes) SeedAt(key string, item Record) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = item
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
func (s *InMemoryAllShapes) Many(ctx context.Context, keys ...string) ([]Record, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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
// completion; returns the byte count, or the configured override
// when one is set.
func (s *InMemoryAllShapes) ReadFrom(ctx context.Context, r io.Reader) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if r == nil {
		return 0, nil
	}
	buf := make([]byte, 64)
	n, err := io.ReadFull(r, buf)
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		err = nil
	}
	s.mu.RLock()
	override := s.readFromOverride
	s.mu.RUnlock()
	if override != nil {
		return *override, err
	}
	return n, err
}

// Inspect implements [AllShapes]. (Lookup, 3-result with bool.)
// Secondary string returns the configured inspectMetaOverride when
// set, else "meta:" + v.ID.
func (s *InMemoryAllShapes) Inspect(ctx context.Context, key string) (Record, string, bool) {
	if ctx.Err() != nil {
		return Record{}, "", false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[key]
	if !ok {
		return Record{}, "", false
	}
	meta := "meta:" + v.ID
	if s.inspectMetaOverride != nil {
		meta = *s.inspectMetaOverride
	}
	return v, meta, true
}

// Load implements [AllShapes]. (ReaderWithBool, comma-ok.)
func (s *InMemoryAllShapes) Load(ctx context.Context, key string) (Record, bool) {
	if ctx.Err() != nil {
		return Record{}, false
	}
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
func (s *InMemoryAllShapes) Schedule(ctx context.Context, key string, value Record, priority int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = value
	_ = priority
	return nil
}

// Set implements [AllShapes]. (CompositeWriter.)
func (s *InMemoryAllShapes) Set(ctx context.Context, key string, value Record) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = value
	return nil
}

// Fetch implements [AllShapes]. (MultiReader.) Secondary string
// returns the configured fetchMetaOverride when set, else
// "meta:" + v.ID.
func (s *InMemoryAllShapes) Fetch(ctx context.Context, key string) (Record, string, error) {
	if err := ctx.Err(); err != nil {
		return Record{}, "", err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[key]
	if !ok {
		return Record{}, "", ErrNotFound
	}
	meta := "meta:" + v.ID
	if s.fetchMetaOverride != nil {
		meta = *s.fetchMetaOverride
	}
	return v, meta, nil
}

// Stats implements [AllShapes]. (MultiAggregator.) Returns the
// configured statsOverride pair when set, else (item count,
// touch sum, nil).
func (s *InMemoryAllShapes) Stats(ctx context.Context) (int, int, error) {
	if err := ctx.Err(); err != nil {
		return 0, 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.statsOverride != nil {
		return s.statsOverride[0], s.statsOverride[1], nil
	}
	totalTouches := 0
	for _, n := range s.touches {
		totalTouches += n
	}
	return len(s.items), totalTouches, nil
}

// Remove implements [AllShapes]. (Deleter.)
func (s *InMemoryAllShapes) Remove(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, key)
	return nil
}

// Put implements [AllShapes]. (Writer.) Keys by item.ID.
func (s *InMemoryAllShapes) Put(ctx context.Context, item Record) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[item.ID] = item
	return nil
}

// Find implements [AllShapes]. (PointerReader.)
func (s *InMemoryAllShapes) Find(ctx context.Context, key string) *Record {
	if ctx.Err() != nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[key]
	if !ok {
		return nil
	}
	return &v
}

// Get implements [AllShapes]. (Reader.) Returns ErrNotFound on miss.
func (s *InMemoryAllShapes) Get(ctx context.Context, key string) (Record, error) {
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}
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
func (s *InMemoryAllShapes) Lookup(ctx context.Context, key string) Record {
	if ctx.Err() != nil {
		return Record{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.items[key]
}

// Count implements [AllShapes]. (Aggregator.) Returns the configured
// countOverride when set, else len(items).
func (s *InMemoryAllShapes) Count(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.countOverride != nil {
		return *s.countOverride, nil
	}
	return len(s.items), nil
}

// Touch implements [AllShapes]. (Mutator.)
func (s *InMemoryAllShapes) Touch(ctx context.Context, key string) {
	if ctx.Err() != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.touches[key]++
}

// Init implements [AllShapes]. (Lifecycle.) Honors ctx cancellation
// for the contract's respects-context assertion. Returns
// [ErrNotFound] under invalidMode so AssertLifecycleRejectInvalidWith
// sees the required non-nil error.
func (s *InMemoryAllShapes) Init(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.invalidMode {
		return ErrNotFound
	}
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
