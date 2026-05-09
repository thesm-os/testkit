// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"context"
	"errors"
	"iter"
	"sync"
)

// Test fixtures for bench tests. These are local to package bench_test
// and mirror the fixtures in suite_test. Each shape needs a minimal
// concrete type to exercise its bench primitives.
//
// Fixtures used by ConcurrentThroughput primitives carry a sync.Mutex
// so they're safe under b.RunParallel; read-only fixtures rely on
// immutability after construction.

var errNotFound = errors.New("not found")

// --- Reader fixture ---

type mapReader struct {
	data map[string]string
}

func newMapReader(data map[string]string) *mapReader {
	return &mapReader{data: data}
}

func (r *mapReader) Get(_ context.Context, key string) (string, error) {
	v, ok := r.data[key]
	if !ok {
		return "", errNotFound
	}
	return v, nil
}

// --- Writer fixture ---

type mapWriter struct {
	mu   sync.Mutex
	data map[string]string
}

func newMapWriter() *mapWriter {
	return &mapWriter{data: make(map[string]string)}
}

type entry struct {
	Key   string
	Value string
}

func (w *mapWriter) Put(_ context.Context, e entry) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.data[e.Key] = e.Value
	return nil
}

// --- Deleter fixture ---

type delStore struct {
	mu   sync.Mutex
	data map[string]bool
}

func newDelStore() *delStore {
	return &delStore{data: map[string]bool{"existing": true}}
}

func (s *delStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.data[key] {
		return errNotFound
	}
	delete(s.data, key)
	return nil
}

// --- Aggregator fixture ---

type itemCounter struct{ n int }

func newItemCounter(n int) *itemCounter { return &itemCounter{n: n} }

func (c *itemCounter) Count(_ context.Context) (int, error) { return c.n, nil }

// --- Lifecycle fixture ---

type lifecycle struct {
	mu     sync.Mutex
	opened bool
}

func newLifecycle() *lifecycle { return &lifecycle{} }

func (l *lifecycle) Open(_ context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.opened = true
	return nil
}

// --- Pure fixture ---

type counter struct{ n int }

func newCounter() *counter { return &counter{n: 42} }

func (c *counter) Value() int { return c.n }

// --- Predicate fixture ---

type validator struct{ valid bool }

func newValidator(v bool) *validator { return &validator{valid: v} }

func (v *validator) IsValid() bool { return v.valid }

// --- Mutator fixture ---

type accumulator struct {
	mu    sync.Mutex
	total int64
}

func newAccumulator() *accumulator { return &accumulator{} }

func (a *accumulator) Add(_ context.Context, v int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.total += v
}

// --- ReaderWithBool fixture ---

type boolMap struct{ data map[string]int64 }

func newBoolMap(data map[string]int64) *boolMap { return &boolMap{data: data} }

func (m *boolMap) Load(_ context.Context, key string) (int64, bool) {
	v, ok := m.data[key]
	return v, ok
}

// --- Lookup fixture ---

type lookupMeta struct{ Version string }

type lookupStore struct {
	values map[string]int64
	meta   map[string]lookupMeta
}

func newLookupStore() *lookupStore {
	return &lookupStore{
		values: map[string]int64{"a": 10},
		meta:   map[string]lookupMeta{"a": {Version: "v1"}},
	}
}

func (s *lookupStore) Inspect(_ context.Context, key string) (int64, lookupMeta, bool) {
	v, ok := s.values[key]
	if !ok {
		return 0, lookupMeta{}, false
	}
	return v, s.meta[key], true
}

// --- PoisonAccessor fixture ---

type healthChecker struct{ err error }

func newHealthChecker() *healthChecker { return &healthChecker{} }

func (h *healthChecker) Err() error { return h.err }

// --- Stream fixture ---

type listStore struct {
	items []string
}

func (s *listStore) List() iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		for _, item := range s.items {
			if !yield(item, nil) {
				return
			}
		}
	}
}

// --- BatchReader fixture ---

type batchStore struct {
	data map[string]string
}

func newBatchStore(data map[string]string) *batchStore {
	return &batchStore{data: data}
}

func (s *batchStore) GetMany(_ context.Context, keys []string) ([]string, error) {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		v, ok := s.data[k]
		if !ok {
			return nil, errNotFound
		}
		out = append(out, v)
	}
	return out, nil
}

// --- CompositeWriter fixture ---

type nsStore struct {
	mu   sync.Mutex
	data map[string][]string
}

func newNsStore() *nsStore {
	return &nsStore{data: make(map[string][]string)}
}

func (s *nsStore) Put(_ context.Context, ns, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[ns] = append(s.data[ns], value)
	return nil
}

// --- MultiAggregator fixture ---

type stats struct {
	count int
	total int64
}

func newStats(count int, total int64) *stats { return &stats{count: count, total: total} }

func (s *stats) Stats(_ context.Context) (int, int64, error) {
	return s.count, s.total, nil
}

// --- MultiArgWriter fixture ---

type subscriber struct {
	mu   sync.Mutex
	subs map[string]struct{}
}

func newSubscriber() *subscriber {
	return &subscriber{subs: make(map[string]struct{})}
}

func (s *subscriber) Subscribe(_ context.Context, topic, _ string, _ int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subs[topic] = struct{}{}
	return nil
}

// --- MultiReader fixture ---

type metaStore struct {
	values map[string]string
	metas  map[string]int
}

func newMetaStore() *metaStore {
	return &metaStore{
		values: map[string]string{"a": "alpha"},
		metas:  map[string]int{"a": 1},
	}
}

func (s *metaStore) Inspect(_ context.Context, k string) (string, int, error) {
	v, ok := s.values[k]
	if !ok {
		return "", 0, errNotFound
	}
	return v, s.metas[k], nil
}

// --- PointerReader fixture ---

type ptrStore struct {
	data map[string]*string
}

func newPtrStore() *ptrStore {
	v := "alpha"
	return &ptrStore{data: map[string]*string{"a": &v}}
}

func (s *ptrStore) Find(_ context.Context, k string) *string {
	return s.data[k]
}

// --- ReaderNoError fixture ---

type infallibleStore struct {
	data map[string]string
}

func newInfallibleStore(data map[string]string) *infallibleStore {
	return &infallibleStore{data: data}
}

func (s *infallibleStore) Get(_ context.Context, k string) string {
	return s.data[k]
}

// --- StreamConsumer fixture ---

// chanStream is a fixed-size sequence used to exercise the
// StreamConsumer shape. The consumer sums the values; a nil source
// produces an error via the chanConsumer.
type chanStream []int

type chanConsumer struct{}

func newChanConsumer() *chanConsumer { return &chanConsumer{} }

func (*chanConsumer) Sum(_ context.Context, src chanStream) (int, error) {
	if src == nil {
		return 0, errNotFound
	}
	sum := 0
	for _, v := range src {
		sum += v
	}
	return sum, nil
}

// --- VoidLifecycle fixture ---

type resetter struct {
	mu sync.Mutex
	n  int
}

func newResetter() *resetter { return &resetter{} }

func (r *resetter) Reset(_ context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.n = 0
}
