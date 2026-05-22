// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/bindings"
	"go.thesmos.sh/testkit/suite"
)

type cache struct{ data map[string]string }

func newCache(data map[string]string) *cache { return &cache{data: data} }

// Lookup is infallible — missing keys return the zero value (empty string).
func (c *cache) Lookup(_ context.Context, k string) string { return c.data[k] }

func readerNoErrorCtx(t *testing.T, data map[string]string) suite.ReaderNoErrorContext[*cache, string, string] {
	t.Helper()
	return suite.ReaderNoErrorContext[*cache, string, string]{
		T: t,
		ReaderNoErrorBindings: bindings.ReaderNoErrorBindings[*cache, string, string]{
			Factory: func() *cache { return newCache(data) },
			Call: func(ctx context.Context, c *cache, k string) string {
				return c.Lookup(ctx, k)
			},
		},
	}
}

func TestReaderNoError(t *testing.T) {
	t.Parallel()
	data := map[string]string{"k1": "alpha"}

	t.Run("ReturnsForKey surfaces the value", func(t *testing.T) {
		t.Parallel()
		suite.AssertReaderNoErrorReturnsForKey[*cache, string, string](
			"k1", "alpha",
		)(readerNoErrorCtx(t, data))
	})

	t.Run("Consistent yields equal values across N calls", func(t *testing.T) {
		t.Parallel()
		suite.AssertReaderNoErrorConsistent[*cache, string, string](
			"k1", 4,
		)(readerNoErrorCtx(t, data))
	})

	t.Run("ZeroOnUnknown returns the zero value for missing keys", func(t *testing.T) {
		t.Parallel()
		suite.AssertReaderNoErrorZeroOnUnknown[*cache, string, string](
			"missing", "",
		)(readerNoErrorCtx(t, data))
	})

	t.Run("RespectsContext is a panic-free smoke", func(t *testing.T) {
		t.Parallel()
		suite.AssertReaderNoErrorRespectsContext[*cache, string, string](
			"k1",
		)(readerNoErrorCtx(t, data))
	})

	t.Run("ConcurrentSafe runs without races", func(t *testing.T) {
		t.Parallel()
		suite.AssertReaderNoErrorConcurrentSafe[*cache, string, string](
			"k1", 4, 50,
		)(readerNoErrorCtx(t, data))
	})
}

func TestAssertReaderNoErrorBaseline(t *testing.T) {
	t.Parallel()
	data := map[string]string{"k1": "alpha"}
	suite.AssertReaderNoErrorBaseline[*cache, string, string](
		"k1", "alpha", "missing", "",
	)(readerNoErrorCtx(t, data))
}
