// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"iter"
	"testing"

	"go.thesmos.sh/testkit/bindings"
	"go.thesmos.sh/testkit/suite"
)

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

func streamCtx(t *testing.T, items []string) suite.StreamContext[*listStore, string] {
	t.Helper()
	return suite.StreamContext[*listStore, string]{
		T: t,
		StreamBindings: bindings.StreamBindings[*listStore, string]{
			Factory: func() *listStore { return &listStore{items: items} },
			Call: func(_ context.Context, s *listStore) iter.Seq2[string, error] {
				return s.List()
			},
		},
	}
}

func TestStream(t *testing.T) {
	t.Parallel()

	t.Run("Completes drains every item with no terminal error", func(t *testing.T) {
		t.Parallel()
		suite.AssertStreamCompletes[*listStore, string]()(
			streamCtx(t, []string{"a", "b", "c"}))
	})

	t.Run("RespectsBreak stops yielding when the consumer breaks", func(t *testing.T) {
		t.Parallel()
		suite.AssertStreamRespectsBreak[*listStore, string]()(
			streamCtx(t, []string{"a", "b", "c"}))
	})

	t.Run("Reentrant supports multiple iterations of the same stream", func(t *testing.T) {
		t.Parallel()
		suite.AssertStreamReentrant[*listStore, string]()(
			streamCtx(t, []string{"a", "b"}))
	})

	t.Run("YieldsInOrder asserts the consumer-supplied predicate across pairs", func(t *testing.T) {
		t.Parallel()
		suite.AssertStreamYieldsInOrder[*listStore, string](
			func(a, b string) bool { return a < b },
		)(streamCtx(t, []string{"a", "b", "c"}))
	})

	t.Run("HasNoDuplicates asserts no two items share the same key", func(t *testing.T) {
		t.Parallel()
		suite.AssertStreamHasNoDuplicates[*listStore, string, string](
			func(v string) string { return v },
		)(streamCtx(t, []string{"a", "b", "c"}))
	})

	t.Run("RespectsContext stops yielding after mid-stream cancel", func(t *testing.T) {
		t.Parallel()
		// listStore.List ignores ctx; wire a Call adapter that wraps
		// the iterator with ctx.Err() polling between yields.
		ctx := suite.StreamContext[*listStore, string]{
			T: t,
			StreamBindings: bindings.StreamBindings[*listStore, string]{
				Factory: func() *listStore { return &listStore{items: []string{"a", "b", "c", "d"}} },
				Call: func(c context.Context, s *listStore) iter.Seq2[string, error] {
					return func(yield func(string, error) bool) {
						for _, item := range s.items {
							if err := c.Err(); err != nil {
								yield("", err)
								return
							}
							if !yield(item, nil) {
								return
							}
						}
					}
				},
			},
		}
		suite.AssertStreamRespectsContext[*listStore, string]()(ctx)
	})

	t.Run("ConcurrentSafe runs without races under N workers", func(t *testing.T) {
		t.Parallel()
		suite.AssertStreamConcurrentSafe[*listStore, string](4)(
			streamCtx(t, []string{"a", "b", "c"}))
	})
}
