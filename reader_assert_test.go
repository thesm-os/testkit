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
		Call: func(ctx context.Context, r *mapReader, k string) (string, error) {
			return r.Get(ctx, k)
		},
	}
}

func TestAssertReturnsForKey(t *testing.T) {
	t.Parallel()
	ctx := readerCtx(t, map[string]string{"a": "alpha"})
	testkit.AssertReturnsForKey[*mapReader, string, string]("a", "alpha")(ctx)
}

func TestAssertReturnsSentinel(t *testing.T) {
	t.Parallel()
	ctx := readerCtx(t, map[string]string{})
	testkit.AssertReturnsSentinel[*mapReader, string, string]("missing", errNotFound)(ctx)
}

func TestAssertConsistentReads(t *testing.T) {
	t.Parallel()
	ctx := readerCtx(t, map[string]string{"x": "value"})
	testkit.AssertConsistentReads[*mapReader, string, string]("x", 5)(ctx)
}

func TestAssertReadsAreNonMutating(t *testing.T) {
	t.Parallel()
	ctx := readerCtx(t, map[string]string{"x": "value"})
	testkit.AssertReadsAreNonMutating[*mapReader, string, string, int](
		"x",
		func(_ context.Context, r *mapReader) int { return len(r.data) },
	)(ctx)
}

func TestAssertReaderConcurrentSafe(t *testing.T) {
	t.Parallel()
	ctx := readerCtx(t, map[string]string{"x": "value"})
	testkit.AssertReaderConcurrentSafe[*mapReader, string, string]("x", 4, 100)(ctx)
}
