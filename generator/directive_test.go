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

func TestRenderMethodDirectives(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when no method carries a directive", func(t *testing.T) {
		t.Parallel()
		methods := []MethodInfo{{Name: "Get"}, {Name: "Put"}}
		testkit.True(t, RenderMethodDirectives(methods) == nil,
			"empty directive set short-circuits")
	})

	t.Run("renders aligned method-prefixed lines with consumer annotation", func(t *testing.T) {
		t.Parallel()
		// `errors` is registered with Consumed("stub", "...").
		methods := []MethodInfo{
			{Name: "Get", Directives: []directive.Directive{
				{Name: directive.Errors, Args: []string{"ErrNotFound"}},
			}},
			{Name: "Submit", Directives: []directive.Directive{
				{Name: directive.Errors, Args: []string{"ErrConflict"}},
			}},
		}
		got := RenderMethodDirectives(methods)
		testkit.Len(t, got, 2, "one line per directive")
		testkit.Assert(t, got[0]).
			HasPrefix("Get:    //testkit:errors ErrNotFound", "name padded to widest").
			Contains("[stub:", "consumer annotation present")
		testkit.Assert(t, got[1]).HasPrefix("Submit: //testkit:errors ErrConflict",
			"longest name fits without padding")
	})

	t.Run("omits annotation when descriptor declares no consumers", func(t *testing.T) {
		t.Parallel()
		// `ctx` exists in DefaultRegistry but declares no
		// Consumed(...) — line renders without the [..] suffix.
		methods := []MethodInfo{
			{Name: "Get", Directives: []directive.Directive{
				{Name: directive.Ctx},
			}},
		}
		got := RenderMethodDirectives(methods)
		testkit.Len(t, got, 1, "single line")
		testkit.Assert(t, got[0]).
			HasPrefix("Get: //testkit:ctx", "directive name").
			NotContains("[", "no consumer annotation")
	})
}

func TestConsumerAnnotations(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for unknown directive", func(t *testing.T) {
		t.Parallel()
		got := consumerAnnotations(directive.DefaultRegistry(), "nonexistent")
		testkit.True(t, got == nil, "unknown → nil")
	})

	t.Run("returns nil for directive without Consumers", func(t *testing.T) {
		t.Parallel()
		// `ctx` is known but has no Consumed(...) entries.
		got := consumerAnnotations(directive.DefaultRegistry(), directive.Ctx)
		testkit.True(t, got == nil, "no consumers → nil")
	})

	t.Run("preferred-order entries surface ahead of others", func(t *testing.T) {
		t.Parallel()
		// Custom registry exercises the rest path: a consumer key
		// outside the preferred {stub,suite,bench,model} list must
		// trail the preferred entries, sorted alphabetically.
		r := directive.NewRegistry()
		r.MustRegister(directive.New(
			"custom", directive.InCategory(directive.Mixin),
			directive.Consumed("zsim", "scenario hook"),
			directive.Consumed("apex", "alpha hook"),
			directive.Consumed("suite", "primary"),
			directive.Consumed("stub", "fault hook"),
		))
		got := consumerAnnotations(r, "custom")
		testkit.Len(t, got, 4, "four entries")
		testkit.Equal(t, got[0], "stub: fault hook", "preferred[0] = stub")
		testkit.Equal(t, got[1], "suite: primary", "preferred[1] = suite")
		// "apex" sorts before "zsim" alphabetically among the rest.
		testkit.Equal(t, got[2], "apex: alpha hook", "rest sorted alphabetically")
		testkit.Equal(t, got[3], "zsim: scenario hook", "rest sorted alphabetically")
	})
}
