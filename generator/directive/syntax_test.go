// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directive_test

import (
	"slices"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
)

// tokensEqual reports whether two token slices match exactly. Used by
// every parser test in this package; keeping it private to syntax_test
// avoids leaking testing-internal helpers across files.
func tokensEqual(a, b []directive.Token) bool {
	return slices.EqualFunc(a, b, func(ta, tb directive.Token) bool {
		return ta.Name == tb.Name && ta.Off == tb.Off && slices.Equal(ta.Args, tb.Args)
	})
}

func TestParseLine(t *testing.T) {
	t.Parallel()

	t.Run("empty input returns nil", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, directive.ParseLine("") == nil, "empty body")
		testkit.True(t, directive.ParseLine("   ") == nil, "whitespace-only body")
	})

	t.Run("standalone form: no args", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, tokensEqual(directive.ParseLine("atomic"),
			[]directive.Token{{Name: "atomic"}}), "single name")
	})

	t.Run("standalone form: positional args", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, tokensEqual(directive.ParseLine("errors ErrA ErrB"),
			[]directive.Token{{Name: "errors", Args: []string{"ErrA", "ErrB"}}}),
			"errors ErrA ErrB")
	})

	t.Run("bundle form expands every spec into its own token", func(t *testing.T) {
		t.Parallel()
		body := "directive conservative atomic idempotent writer=off timeout=1s errors=ErrA,ErrB"
		got := directive.ParseLine(body)
		want := []directive.Token{
			{Name: "conservative"},
			{Name: "atomic"},
			{Name: "idempotent"},
			{Name: "writer", Off: true},
			{Name: "timeout", Args: []string{"1s"}},
			{Name: "errors", Args: []string{"ErrA", "ErrB"}},
		}
		testkit.True(t, tokensEqual(got, want), "bundle parse")
	})

	t.Run("bundle with only the keyword returns no tokens", func(t *testing.T) {
		t.Parallel()
		testkit.Len(t, directive.ParseLine("directive"), 0, "lone keyword")
	})

	t.Run("bundle preserves Token re-export for parent package callers", func(t *testing.T) {
		t.Parallel()
		// directive.Token / directive.ParseLine are the public surface;
		// this guard exists so renames surface as compile errors.
		_ = directive.Token{Name: "x"}
	})
}
