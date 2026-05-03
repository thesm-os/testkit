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

func TestAssertStreamCompletes(t *testing.T) {
	t.Parallel()
	ctx := streamCtx(t, []string{"a", "b", "c"})
	suite.AssertStreamCompletes[*listStore, string]()(ctx)
}

func TestAssertStreamRespectsBreak(t *testing.T) {
	t.Parallel()
	ctx := streamCtx(t, []string{"a", "b", "c"})
	suite.AssertStreamRespectsBreak[*listStore, string]()(ctx)
}

func TestAssertStreamReentrant(t *testing.T) {
	t.Parallel()
	ctx := streamCtx(t, []string{"a", "b"})
	suite.AssertStreamReentrant[*listStore, string]()(ctx)
}

func TestAssertStreamYieldsInOrder(t *testing.T) {
	t.Parallel()
	ctx := streamCtx(t, []string{"a", "b", "c"})
	suite.AssertStreamYieldsInOrder[*listStore, string](
		func(a, b string) bool { return a < b },
	)(ctx)
}

func TestAssertStreamHasNoDuplicates(t *testing.T) {
	t.Parallel()
	ctx := streamCtx(t, []string{"a", "b", "c"})
	suite.AssertStreamHasNoDuplicates[*listStore, string, string](
		func(v string) string { return v },
	)(ctx)
}
