// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"context"
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
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

func readerCtx(t *testing.T, data map[string]string) testkit.ReaderContext[*mapReader, string, string] {
	t.Helper()
	return testkit.ReaderContext[*mapReader, string, string]{
		T:       t,
		Factory: func() *mapReader { return newMapReader(data) },
		Call: func(r *mapReader, ctx context.Context, k string) (string, error) {
			return r.Get(ctx, k)
		},
	}
}

func TestAssertReturnsForKey(t *testing.T) {
	t.Parallel()

	t.Run("passes when Want matches", func(t *testing.T) {
		t.Parallel()
		ctx := readerCtx(t, map[string]string{"a": "alpha", "b": "beta"})
		ctx.Want = map[string]string{"a": "alpha", "b": "beta"}
		testkit.AssertReturnsForKey[*mapReader, string, string]()(ctx)
	})

	t.Run("skips when Want is empty", func(t *testing.T) {
		t.Parallel()
		ctx := readerCtx(t, map[string]string{"a": "alpha"})
		testkit.AssertReturnsForKey[*mapReader, string, string]()(ctx)
		// Subtest "returns for key" is registered but skipped.
	})
}

func TestAssertReturnsSentinel(t *testing.T) {
	t.Parallel()

	t.Run("passes when unknown key returns sentinel", func(t *testing.T) {
		t.Parallel()
		ctx := readerCtx(t, map[string]string{})
		ctx.Unknown = []string{"missing-key"}
		testkit.AssertReturnsSentinel[*mapReader, string, string](errNotFound)(ctx)
	})

	t.Run("skips when Unknown is empty", func(t *testing.T) {
		t.Parallel()
		ctx := readerCtx(t, map[string]string{})
		testkit.AssertReturnsSentinel[*mapReader, string, string](errNotFound)(ctx)
	})
}

func TestAssertConsistentReads(t *testing.T) {
	t.Parallel()

	t.Run("passes when reads are consistent", func(t *testing.T) {
		t.Parallel()
		ctx := readerCtx(t, map[string]string{"x": "value"})
		ctx.Known = []string{"x"}
		testkit.AssertConsistentReads[*mapReader, string, string](5)(ctx)
	})

	t.Run("skips when Known is empty", func(t *testing.T) {
		t.Parallel()
		ctx := readerCtx(t, map[string]string{})
		testkit.AssertConsistentReads[*mapReader, string, string](5)(ctx)
	})
}

func TestAssertReadsAreNonMutating(t *testing.T) {
	t.Parallel()

	t.Run("passes when read does not mutate", func(t *testing.T) {
		t.Parallel()
		ctx := readerCtx(t, map[string]string{"x": "value"})
		ctx.Known = []string{"x"}
		testkit.AssertReadsAreNonMutating[*mapReader, string, string, int](
			func(r *mapReader) int { return len(r.data) },
		)(ctx)
	})

	t.Run("skips when Known is empty", func(t *testing.T) {
		t.Parallel()
		ctx := readerCtx(t, map[string]string{})
		testkit.AssertReadsAreNonMutating[*mapReader, string, string, int](
			func(r *mapReader) int { return len(r.data) },
		)(ctx)
	})
}

func TestAssertReaderConcurrentSafe(t *testing.T) {
	t.Parallel()

	t.Run("passes with concurrent reads", func(t *testing.T) {
		t.Parallel()
		ctx := readerCtx(t, map[string]string{"x": "value"})
		ctx.Known = []string{"x"}
		testkit.AssertReaderConcurrentSafe[*mapReader, string, string](4, 100)(ctx)
	})

	t.Run("skips when Known is empty", func(t *testing.T) {
		t.Parallel()
		ctx := readerCtx(t, map[string]string{})
		testkit.AssertReaderConcurrentSafe[*mapReader, string, string](4, 100)(ctx)
	})
}
