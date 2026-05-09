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

type mapReader struct {
	data map[string]string
}

var errNotFound = errors.New("not found")

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

func readerCtx(t *testing.T, data map[string]string) suite.ReaderContext[*mapReader, string, string] {
	t.Helper()
	return suite.ReaderContext[*mapReader, string, string]{
		T: t,
		ReaderBindings: bindings.ReaderBindings[*mapReader, string, string]{
			Factory: func() *mapReader { return newMapReader(data) },
			Call: func(ctx context.Context, r *mapReader, k string) (string, error) {
				return r.Get(ctx, k)
			},
		},
	}
}

func TestReader(t *testing.T) {
	t.Parallel()

	t.Run("ReturnsForKey surfaces the value for a known key", func(t *testing.T) {
		t.Parallel()
		ctx := readerCtx(t, map[string]string{"a": "alpha"})
		suite.AssertReturnsForKey[*mapReader, string, string]("a", "alpha")(ctx)
	})

	t.Run("ReturnsSentinel surfaces the configured error for unknown keys", func(t *testing.T) {
		t.Parallel()
		ctx := readerCtx(t, map[string]string{})
		suite.AssertReturnsSentinel[*mapReader, string, string]("missing", errNotFound)(ctx)
	})

	t.Run("ConsistentReads yields equal values across N calls", func(t *testing.T) {
		t.Parallel()
		ctx := readerCtx(t, map[string]string{"x": "value"})
		suite.AssertConsistentReads[*mapReader, string, string]("x", 5)(ctx)
	})

	t.Run("ReadsAreNonMutating leaves observable state unchanged", func(t *testing.T) {
		t.Parallel()
		ctx := readerCtx(t, map[string]string{"x": "value"})
		suite.AssertReadsAreNonMutating[*mapReader, string, string, int](
			"x",
			func(_ context.Context, r *mapReader) int { return len(r.data) },
		)(ctx)
	})

	t.Run("RespectsContext surfaces ctx.Canceled on cancelled call", func(t *testing.T) {
		t.Parallel()
		ctx := suite.ReaderContext[*mapReader, string, string]{
			T: t,
			ReaderBindings: bindings.ReaderBindings[*mapReader, string, string]{
				Factory: func() *mapReader { return newMapReader(map[string]string{"x": "value"}) },
				Call: func(c context.Context, _ *mapReader, _ string) (string, error) {
					return "", c.Err()
				},
			},
		}
		suite.AssertReaderRespectsContext[*mapReader, string, string]("x")(ctx)
	})

	t.Run("ConcurrentSafe runs without races", func(t *testing.T) {
		t.Parallel()
		ctx := readerCtx(t, map[string]string{"x": "value"})
		suite.AssertReaderConcurrentSafe[*mapReader, string, string]("x", 4, 100)(ctx)
	})
}
