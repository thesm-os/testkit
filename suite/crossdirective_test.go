// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"go.thesmos.sh/testkit/suite"
)

// crossDirStore is a directive-flow fixture: a String-keyed map
// with explicit (K, V) write, key-only delete, key-only read, and
// a closed flag. Distinct from existing suite test fixtures so we
// don't conflate the cross-method-invariant flow with the older
// extractKey-based AssertReadAfterWrite tests.
type crossDirStore struct {
	mu     sync.Mutex
	items  map[string]int
	closed bool
}

func newCrossDirStore() *crossDirStore { return &crossDirStore{items: make(map[string]int)} }

var (
	errCDNotFound = errors.New("crossDirStore: not found")
	errCDClosed   = errors.New("crossDirStore: closed")
)

func (s *crossDirStore) Set(_ context.Context, k string, v int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errCDClosed
	}
	s.items[k] = v
	return nil
}

func (s *crossDirStore) Get(_ context.Context, k string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, errCDClosed
	}
	v, ok := s.items[k]
	if !ok {
		return 0, errCDNotFound
	}
	return v, nil
}

func (s *crossDirStore) Delete(_ context.Context, k string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, k)
	return nil
}

func (s *crossDirStore) Put(_ context.Context, v int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items["x"] = v
	return nil
}

func (s *crossDirStore) Scan(_ context.Context) ([]int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]int, 0, len(s.items))
	for _, v := range s.items {
		out = append(out, v)
	}
	return out, nil
}

func (s *crossDirStore) Close(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func (s *crossDirStore) Merge(_ context.Context, v int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[string(rune('a'+v%26))] = v
	return nil
}

func TestAssertReadAfterWriteByKey(t *testing.T) {
	t.Parallel()
	suite.AssertReadAfterWriteByKey[*crossDirStore, string, int](
		"k", 42,
		func(ctx context.Context, impl *crossDirStore, k string, v int) error {
			return impl.Set(ctx, k, v)
		},
		func(ctx context.Context, impl *crossDirStore, k string) (int, error) {
			return impl.Get(ctx, k)
		},
	)(t, newCrossDirStore)
}

func TestAssertDeleteRemovesByKey(t *testing.T) {
	t.Parallel()
	factory := func() *crossDirStore {
		s := newCrossDirStore()
		s.items["k"] = 1
		return s
	}
	suite.AssertDeleteRemovesByKey[*crossDirStore, string, int](
		"k", errCDNotFound,
		func(ctx context.Context, impl *crossDirStore, k string) error {
			return impl.Delete(ctx, k)
		},
		func(ctx context.Context, impl *crossDirStore, k string) (int, error) {
			return impl.Get(ctx, k)
		},
	)(t, factory)
}

func TestAssertStreamReflectsValueWritten(t *testing.T) {
	t.Parallel()
	suite.AssertStreamReflectsValueWritten[*crossDirStore, int](
		7,
		func(ctx context.Context, impl *crossDirStore, v int) error {
			return impl.Put(ctx, v)
		},
		func(ctx context.Context, impl *crossDirStore) ([]int, error) {
			return impl.Scan(ctx)
		},
	)(t, newCrossDirStore)
}

func TestAssertLifecycleAfterClose(t *testing.T) {
	t.Parallel()
	suite.AssertLifecycleAfterClose[*crossDirStore, string, int](
		"k", errCDClosed,
		func(ctx context.Context, impl *crossDirStore) error {
			return impl.Close(ctx)
		},
		func(ctx context.Context, impl *crossDirStore, k string) (int, error) {
			return impl.Get(ctx, k)
		},
	)(t, newCrossDirStore)
}

// closeable is a fixture that tracks closure and has a reflective reader.
type closeable struct {
	mu     sync.Mutex
	closed bool
}

func closeableFactory() *closeable { return &closeable{} }

// Close is the lifecycle method.
func (c *closeable) Close(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

// Count is the aggregator-like reader invoked via reflection. It returns
// an error after close.
func (c *closeable) Count(_ context.Context) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return 0, errors.New("closed")
	}
	return 0, nil
}

func TestAssertLifecycleAfterCloseReflective(t *testing.T) {
	t.Parallel()
	closeFn := func(ctx context.Context, c *closeable) error {
		return c.Close(ctx)
	}
	assertion := suite.AssertLifecycleAfterCloseReflective[*closeable](
		"Count", closeFn,
	)
	assertion(t, closeableFactory)
}

func TestAssertCRDTMerge(t *testing.T) {
	t.Parallel()
	suite.AssertCRDTMerge[*crossDirStore, int](
		1, 2,
		func(ctx context.Context, impl *crossDirStore, v int) error {
			return impl.Merge(ctx, v)
		},
		func(a, b *crossDirStore) bool {
			a.mu.Lock()
			defer a.mu.Unlock()
			b.mu.Lock()
			defer b.mu.Unlock()
			if len(a.items) != len(b.items) {
				return false
			}
			for k, v := range a.items {
				if b.items[k] != v {
					return false
				}
			}
			return true
		},
	)(t, newCrossDirStore)
}
