// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package witness_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/node"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/witness"
)

// The witnesses are what lets a check for a generic subject exist at all, so
// what matters is which constraints yield one and that the answer is
// all-or-nothing.
func TestFor(t *testing.T) {
	t.Parallel()

	t.Run("derives one per parameter", func(t *testing.T) {
		t.Parallel()
		testkit.Len(t, witness.For(params("comparable", "any")), 2, "both parameters take a witness")
	})

	t.Run("derives distinct witnesses per position", func(t *testing.T) {
		t.Parallel()
		// Identical witnesses would let a template that crossed two type
		// parameters typecheck, which is the mistake worth catching and the one
		// no assertion can see.
		got := witness.Args(params("comparable", "any"))
		testkit.Equal(t, got, "[string, int]", "positions take different types")
	})

	t.Run("accepts an explicitly written any", func(t *testing.T) {
		t.Parallel()
		// IsAny holds only for a parameter with no bound written at all; one
		// written `[V any]` carries `any` as an embedded bound.
		testkit.Len(t, witness.For(params("any")), 1, "a written any is still any")
	})

	t.Run("accepts the long spelling of any", func(t *testing.T) {
		t.Parallel()
		testkit.Len(t, witness.For(params("interface{}")), 1, "interface{} is any")
	})

	t.Run("declines a constraint it cannot read", func(t *testing.T) {
		t.Parallel()
		// A named constraint is a reference into a package never loaded, so
		// guessing its type set would produce code that fails to compile for a
		// reason the author could not have predicted.
		testkit.Len(t, witness.For(params("Ordered")), 0, "an unreadable bound yields nothing")
	})

	t.Run("declines the whole list when one parameter fails", func(t *testing.T) {
		t.Parallel()
		// An entry point instantiates the list at once, so a witness for one
		// parameter is worth nothing without one for the rest.
		testkit.Len(t, witness.For(params("any", "Ordered")), 0, "derivation is all-or-nothing")
	})

	t.Run("declines an arity past the palette", func(t *testing.T) {
		t.Parallel()
		// Wrapping the list would hand two parameters the same type and lose
		// exactly the distinctness the palette exists for.
		wide := make([]string, len(witness.Palette)+1)
		for i := range wide {
			wide[i] = "any"
		}
		testkit.Len(t, witness.For(params(wide...)), 0, "an over-long list yields nothing")
	})

	t.Run("yields nothing for a subject that is not generic", func(t *testing.T) {
		t.Parallel()
		testkit.Len(t, witness.For(nil), 0, "an unparameterised subject instantiates nothing")
	})
}

// Args is what a template appends, so its empty form matters as much as its
// populated one.
func TestArgs(t *testing.T) {
	t.Parallel()

	t.Run("renders nothing for a subject that is not generic", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, witness.Args(nil), "", "a call site can append it unconditionally")
	})

	t.Run("renders nothing when no witness could be derived", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, witness.Args(params("Ordered")), "", "an unwitnessable subject renders nothing")
	})
}

// params builds a type-parameter list, one per written bound.
func params(bounds ...string) []*node.TypeParam {
	out := make([]*node.TypeParam, len(bounds))
	for i, b := range bounds {
		c := storefixture.Constraint(storefixture.Named(b))
		c.Raw = b
		out[i] = &node.TypeParam{Name: string(rune('K' + i)), Constraint: c}
	}
	return out
}
