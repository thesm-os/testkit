// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"errors"
	"testing"

	"go.thesmos.sh/testkit/bindings"
	"go.thesmos.sh/testkit/suite"
)

type batchStore struct{ data map[string]string }

var errBatchUnknown = errors.New("batch: unknown key")

func newBatchStore(data map[string]string) *batchStore { return &batchStore{data: data} }

func (s *batchStore) Many(_ context.Context, keys []string) ([]string, error) {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		v, ok := s.data[k]
		if !ok {
			return nil, errBatchUnknown
		}
		out = append(out, v)
	}
	return out, nil
}

func batchReaderCtx(t *testing.T, data map[string]string) suite.BatchReaderContext[*batchStore, string, string] {
	t.Helper()
	return suite.BatchReaderContext[*batchStore, string, string]{
		T: t,
		BatchReaderBindings: bindings.BatchReaderBindings[*batchStore, string, string]{
			Factory: func() *batchStore { return newBatchStore(data) },
			Call: func(ctx context.Context, s *batchStore, keys []string) ([]string, error) {
				return s.Many(ctx, keys)
			},
		},
	}
}

func TestBatchReader(t *testing.T) {
	t.Parallel()

	t.Run("ReturnsAll surfaces every key's value", func(t *testing.T) {
		t.Parallel()
		ctx := batchReaderCtx(t, map[string]string{"a": "alpha", "b": "beta"})
		suite.AssertBatchReaderReturnsAll[*batchStore, string, string](
			[]string{"a", "b"}, []string{"alpha", "beta"})(ctx)
	})

	t.Run("ReturnsSentinel surfaces the configured error for unknown keys", func(t *testing.T) {
		t.Parallel()
		ctx := batchReaderCtx(t, map[string]string{"a": "alpha"})
		suite.AssertBatchReaderReturnsSentinel[*batchStore, string, string](
			[]string{"a", "missing"}, errBatchUnknown)(ctx)
	})

	t.Run("Consistent yields equal-length slices across N calls", func(t *testing.T) {
		t.Parallel()
		ctx := batchReaderCtx(t, map[string]string{"a": "alpha", "b": "beta"})
		suite.AssertBatchReaderConsistent[*batchStore, string, string](
			[]string{"a", "b"}, 4)(ctx)
	})

	t.Run("RespectsContext surfaces ctx.Canceled on cancelled call", func(t *testing.T) {
		t.Parallel()
		// Impl that surfaces ctx.Err immediately — required to observe
		// the contract under a pre-cancelled context.
		ctx := suite.BatchReaderContext[*batchStore, string, string]{
			T: t,
			BatchReaderBindings: bindings.BatchReaderBindings[*batchStore, string, string]{
				Factory: func() *batchStore { return newBatchStore(map[string]string{"a": "alpha"}) },
				Call: func(c context.Context, _ *batchStore, _ []string) ([]string, error) {
					if err := c.Err(); err != nil {
						return nil, err
					}
					return nil, nil
				},
			},
		}
		suite.AssertBatchReaderRespectsContext[*batchStore, string, string]([]string{"a"})(ctx)
	})

	t.Run("ConcurrentSafe runs without races under N workers", func(t *testing.T) {
		t.Parallel()
		ctx := batchReaderCtx(t, map[string]string{"a": "alpha", "b": "beta"})
		suite.AssertBatchReaderConcurrentSafe[*batchStore, string, string](
			[]string{"a", "b"}, 4, 50)(ctx)
	})
}

func TestAssertBatchReaderBaseline(t *testing.T) {
	t.Parallel()
	data := map[string]string{"a": "alpha", "b": "beta"}
	ctx := suite.BatchReaderContext[*batchStore, string, string]{
		T: t,
		BatchReaderBindings: bindings.BatchReaderBindings[*batchStore, string, string]{
			Factory: func() *batchStore { return newBatchStore(data) },
			Call: func(c context.Context, s *batchStore, keys []string) ([]string, error) {
				if err := c.Err(); err != nil {
					return nil, err
				}
				return s.Many(c, keys)
			},
		},
	}
	suite.AssertBatchReaderBaseline[*batchStore, string, string](
		[]string{"a", "b"}, []string{"alpha", "beta"})(ctx)
}
