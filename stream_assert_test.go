// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"iter"
	"testing"

	"go.thesmos.sh/testkit"
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

func streamCtx(t *testing.T, items []string) testkit.StreamContext[*listStore, string] {
	t.Helper()
	return testkit.StreamContext[*listStore, string]{
		T:       t,
		Factory: func() *listStore { return &listStore{items: items} },
		Call: func(s *listStore) iter.Seq2[string, error] {
			return s.List()
		},
	}
}

func TestAssertStreamCompletes(t *testing.T) {
	t.Parallel()
	ctx := streamCtx(t, []string{"a", "b", "c"})
	testkit.AssertStreamCompletes[*listStore, string]()(ctx)
}

func TestAssertStreamRespectsBreak(t *testing.T) {
	t.Parallel()
	ctx := streamCtx(t, []string{"a", "b", "c"})
	testkit.AssertStreamRespectsBreak[*listStore, string]()(ctx)
}

func TestAssertStreamReentrant(t *testing.T) {
	t.Parallel()
	ctx := streamCtx(t, []string{"a", "b"})
	testkit.AssertStreamReentrant[*listStore, string]()(ctx)
}

func TestAssertStreamYieldsInOrder(t *testing.T) {
	t.Parallel()
	ctx := streamCtx(t, []string{"a", "b", "c"})
	testkit.AssertStreamYieldsInOrder[*listStore, string](
		func(a, b string) bool { return a < b },
	)(ctx)
}

func TestAssertStreamHasNoDuplicates(t *testing.T) {
	t.Parallel()
	ctx := streamCtx(t, []string{"a", "b", "c"})
	testkit.AssertStreamHasNoDuplicates[*listStore, string, string](
		func(v string) string { return v },
	)(ctx)
}
