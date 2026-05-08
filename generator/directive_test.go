// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator

import (
	"go/token"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
)

func TestParseDirectivesFromDoc(t *testing.T) {
	t.Parallel()

	t.Run("standalone form parses one directive per line", func(t *testing.T) {
		t.Parallel()
		doc := `// Get fetches by key.
//
//testkit:errors ErrNotFound
//testkit:cacheable
`
		pos := token.Position{Filename: "x.go", Line: 1}
		got := parseDirectivesFromDoc(doc, pos)
		testkit.Len(t, got, 2, "two directives")
		testkit.Equal(t, got[0].Name, "errors", "first directive name")
		testkit.Equal(t, got[0].Args[0], "ErrNotFound", "first directive arg")
		testkit.Equal(t, got[1].Name, "cacheable", "second directive name")
	})

	t.Run("bundle form expands into multiple directives", func(t *testing.T) {
		t.Parallel()
		doc := `// Save writes the entity.
//
//testkit:directive atomic idempotent writer=off timeout=1s
`
		pos := token.Position{Filename: "x.go", Line: 5}
		got := parseDirectivesFromDoc(doc, pos)
		testkit.Len(t, got, 4, "bundle expands to 4 directives")
		testkit.Equal(t, got[0].Name, "atomic", "atomic")
		testkit.Equal(t, got[1].Name, "idempotent", "idempotent")
		testkit.Equal(t, got[2].Name, "writer", "writer")
		testkit.True(t, got[2].Off, "writer=off sets Off=true")
		testkit.Equal(t, got[3].Name, "timeout", "timeout")
		testkit.Equal(t, got[3].Args[0], "1s", "timeout arg")
	})

	t.Run("non-testkit comments are ignored", func(t *testing.T) {
		t.Parallel()
		doc := `// regular comment
// another normal line
//testkit:atomic
//some-other-directive: foo
`
		got := parseDirectivesFromDoc(doc, token.Position{})
		testkit.Len(t, got, 1, "only //testkit: lines parse")
		testkit.Equal(t, got[0].Name, "atomic", "atomic")
	})

	t.Run("empty input returns nil", func(t *testing.T) {
		t.Parallel()
		got := parseDirectivesFromDoc("", token.Position{})
		testkit.True(t, got == nil, "empty doc returns nil")
	})
}

// Compile-time guard: directive package is reachable from this test file.
// Without this line `directive` would be unused if tests above were stubbed.
var _ = directive.Errors
