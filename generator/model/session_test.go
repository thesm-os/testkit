// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape"

	"go.thesmos.sh/testkit"
)

// TestVersionMemberRefusal holds version= to the field projection every
// consumer of the stamp performs: a method or a missing member is refused at
// the directive, not left to surface as a build error in the generated
// package that nothing attributes.
func TestVersionMemberRefusal(t *testing.T) {
	t.Parallel()

	t.Run("a field passes and elects the twin", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, sessionStore(t, "Rev"))
		testkit.True(t, b.Reference.Twin(), "the stamped fixture rides the twin floor")
		testkit.Assert(t, b.Reference.TwinWhy).Contains("version member",
			"for the version arm's own reason")
	})

	t.Run("a zero-arg method is refused by name", func(t *testing.T) {
		t.Parallel()
		got := generateBoth(t, sessionStore(t, "Stamp")).Diagnostics()
		testkit.Equal(t, len(got), 1, "one diagnostic")
		testkit.Assert(t, got[0].Message).Contains("names a method",
			"the stamp is read and assigned as a field, and the message says which spelling broke that")
	})

	t.Run("a missing member is refused by name", func(t *testing.T) {
		t.Parallel()
		got := generateBoth(t, sessionStore(t, "Gone")).Diagnostics()
		testkit.Equal(t, len(got), 1, "one diagnostic")
		testkit.Assert(t, got[0].Message).Contains("names no member",
			"a typo dies at the directive rather than in the consumer's build")
	})

	t.Run("an out-of-reach struct passes through", func(t *testing.T) {
		t.Parallel()
		s := sessionStore(t, "Rev")
		// Re-point the reader's value stamp at a declaration the store does
		// not hold: refusing what cannot be seen would break a witnessed
		// value, so the compile keeps this case honest instead.
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name == "Get" {
					shape.MetaValueType.Set(m.EnsureMeta(), "example.com/elsewhere.Value", "test")
				}
			}
		}
		got := generateBoth(t, s).Diagnostics()
		testkit.Equal(t, len(got), 0, "no declaration in reach, no refusal to make")
	})

	// The cas path shares the validator but not the site: the cell assigns
	// the stamp it guards (v.Rev = cur.Rev + 1), so the method form breaks
	// an lvalue there, not merely a projection.
	t.Run("a cas version method is refused before the cell derives", func(t *testing.T) {
		t.Parallel()
		got := generateBoth(t, casStore(t, "Next")).Diagnostics()
		testkit.Equal(t, len(got), 1, "one diagnostic")
		testkit.Assert(t, got[0].Message).Contains("names a method",
			"the cell's stamp is an lvalue, and the refusal happens at the directive")
	})

	t.Run("a cas version field derives the cell", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, casStore(t, "Version"))
		testkit.True(t, b.Reference.IsContract(), "the shipped cell holds")
		testkit.Equal(t, b.Reference.VersionField, "Version", "guarding the stamped field")
	})
}
