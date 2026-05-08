// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator_test

import (
	"go/token"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
)

func TestSourceAttribution(t *testing.T) {
	t.Parallel()

	t.Run("empty positions returns empty string", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, generator.SourceAttribution(nil), "", "no positions")
	})

	t.Run("single line emits filename:line", func(t *testing.T) {
		t.Parallel()
		got := generator.SourceAttribution([]token.Position{{Filename: "/abs/path/to/store.go", Line: 42}})
		testkit.Equal(t, got, "store.go:42", "absolute paths render via base name only")
	})

	t.Run("zero line is ignored", func(t *testing.T) {
		t.Parallel()
		got := generator.SourceAttribution([]token.Position{{Filename: "store.go", Line: 0}})
		testkit.Equal(t, got, "", "line=0 is treated as missing")
	})

	t.Run("range collapses to filename:min-max", func(t *testing.T) {
		t.Parallel()
		got := generator.SourceAttribution([]token.Position{
			{Filename: "store.go", Line: 10},
			{Filename: "store.go", Line: 25},
			{Filename: "store.go", Line: 17},
		})
		testkit.Equal(t, got, "store.go:10-25", "min and max emitted regardless of order")
	})

	t.Run("same line collapses to single value", func(t *testing.T) {
		t.Parallel()
		got := generator.SourceAttribution([]token.Position{
			{Filename: "store.go", Line: 5},
			{Filename: "store.go", Line: 5},
		})
		testkit.Equal(t, got, "store.go:5", "min==max omits range")
	})

	t.Run("empty filename short-circuits", func(t *testing.T) {
		t.Parallel()
		got := generator.SourceAttribution([]token.Position{{Filename: "", Line: 10}})
		testkit.Equal(t, got, "", "filename is required")
	})

	t.Run("first non-empty filename wins", func(t *testing.T) {
		t.Parallel()
		got := generator.SourceAttribution([]token.Position{
			{Filename: "", Line: 0},
			{Filename: "store.go", Line: 7},
		})
		testkit.Equal(t, got, "store.go:7", "skipped entries do not block detection")
	})
}
