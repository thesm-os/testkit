// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ref_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/ref"
)

func TestCursorTable(t *testing.T) {
	t.Parallel()

	intLess := func(a, b int) bool { return a < b }

	t.Run("drains in key order across pages", func(t *testing.T) {
		t.Parallel()
		tab := ref.NewCursorTable[int, string](intLess)
		_ = tab.Put(t.Context(), 3, "c")
		_ = tab.Put(t.Context(), 1, "a")
		_ = tab.Put(t.Context(), 2, "b")
		var drained []string
		cursor := 0
		for range 10 {
			page, next, _ := tab.Page(t.Context(), cursor, 2)
			drained = append(drained, page...)
			if next == 0 {
				break
			}
			cursor = next
		}
		testkit.Equal(t, drained, []string{"a", "b", "c"}, "sorted drain")
	})

	t.Run("zero next-cursor signals end", func(t *testing.T) {
		t.Parallel()
		tab := ref.NewCursorTable[int, string](intLess)
		_ = tab.Put(t.Context(), 1, "a")
		_, next, _ := tab.Page(t.Context(), 0, 5)
		testkit.Equal(t, next, 0, "page drained")
	})

	t.Run("cursor past end returns empty page", func(t *testing.T) {
		t.Parallel()
		tab := ref.NewCursorTable[int, string](intLess)
		_ = tab.Put(t.Context(), 1, "a")
		page, next, _ := tab.Page(t.Context(), 99, 5)
		testkit.Equal(t, len(page), 0, "no items")
		testkit.Equal(t, next, 0, "no next")
	})

	t.Run("negative cursor normalized to zero", func(t *testing.T) {
		t.Parallel()
		tab := ref.NewCursorTable[int, string](intLess)
		_ = tab.Put(t.Context(), 1, "a")
		page, _, _ := tab.Page(t.Context(), -5, 5)
		testkit.Equal(t, page, []string{"a"}, "normalized")
	})

	t.Run("Len reports unique insertions", func(t *testing.T) {
		t.Parallel()
		tab := ref.NewCursorTable[int, string](intLess)
		_ = tab.Put(t.Context(), 1, "a")
		_ = tab.Put(t.Context(), 1, "a2")
		_ = tab.Put(t.Context(), 2, "b")
		testkit.Equal(t, tab.Len(), 2, "two unique keys")
	})
}
