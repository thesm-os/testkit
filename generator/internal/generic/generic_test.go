// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generic_test

import (
	"testing"

	"go.thesmos.sh/eidos/node"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/generic"
)

// The declaration form is what a generated type or function writes between its
// name and its parentheses. Losing a constraint there produces code that
// compiles for the fixture that was in front of the author and fails for the
// one that was not.
func TestParams(t *testing.T) {
	t.Parallel()

	t.Run("carries every parameter and its bound", func(t *testing.T) {
		t.Parallel()
		got := generic.Params(params(bound("comparable"), bound("any")))
		testkit.Len(t, got, 2, "both parameters survive")
		testkit.Equal(t, got[0].Name, "K", "the first keeps its name")
		testkit.Equal(t, got[1].Name, "V", "the second keeps its name")
		testkit.True(t, got[0].Constraint != nil, "the first keeps its bound")
	})

	t.Run("renders no constraint for an any bound", func(t *testing.T) {
		t.Parallel()
		// `any` is the absence of a bound, and emitting it would spell
		// `[K any any]` wherever the backend already supplies the default.
		got := generic.Params([]*node.TypeParam{{Name: "K", Constraint: &node.Constraint{Raw: "any"}}})
		testkit.Len(t, got, 1, "the parameter survives")
		testkit.True(t, got[0].Constraint == nil, "an any bound renders nothing")
	})

	t.Run("declines a declaration carrying none", func(t *testing.T) {
		t.Parallel()
		// nil rather than an empty slice, so a template appending the rendered
		// form does so unconditionally and gets nothing.
		testkit.True(t, generic.Params(nil) == nil, "a non-generic declaration has no parameters")
		testkit.True(t, generic.Params([]*node.TypeParam{}) == nil, "an empty list is the same answer")
	})

	t.Run("keeps a parameter whose bound the source omitted", func(t *testing.T) {
		t.Parallel()
		// `[T any]` and `[T]` are both legal source; dropping the parameter for
		// want of a bound would lose it from the generated signature entirely.
		got := generic.Params(params(nil))
		testkit.Len(t, got, 1, "an unbounded parameter is still a parameter")
		testkit.Equal(t, got[0].Name, "K", "it keeps its name")
	})
}

// The use form is what every generated identifier naming one of the subject's
// own types has to carry — a generic type referenced bare does not compile.
func TestArgs(t *testing.T) {
	t.Parallel()

	t.Run("renders the parameter names in order", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, generic.Args(params(bound("comparable"), bound("any"))), "[K, V]",
			"the use form names the parameters, not their bounds")
	})

	t.Run("renders a single parameter without a separator", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, generic.Args(params(bound("any"))), "[K]", "one parameter needs no comma")
	})

	t.Run("renders nothing for a declaration carrying none", func(t *testing.T) {
		t.Parallel()
		// Empty rather than `[]`, because a template appends this to every
		// identifier it writes and `[]` compiles as nothing legal.
		testkit.Equal(t, generic.Args(nil), "", "a non-generic declaration renders nothing")
		testkit.Equal(t, generic.Args([]*node.TypeParam{}), "", "an empty list is the same answer")
	})
}

// params builds a list named for the positions a key-value subject uses, so a
// test reads which parameter it means rather than counting.
func params(bounds ...*node.Constraint) []*node.TypeParam {
	names := []string{"K", "V", "R"}
	out := make([]*node.TypeParam, len(bounds))
	for i, c := range bounds {
		out[i] = &node.TypeParam{Name: names[i], Constraint: c}
	}
	return out
}

// bound builds a constraint the way a frontend does: the printed source form
// plus the refs it embeds. The embedded list is what carries the meaning —
// a constraint with none reads as `any` however its Raw is spelled.
func bound(raw string) *node.Constraint {
	return &node.Constraint{Raw: raw, Embedded: []*node.TypeRef{{Name: raw}}}
}
