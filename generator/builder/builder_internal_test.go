// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal tests for the parts of the projection a rendered file cannot show.
// An embed the frontend recorded without a usable type is dropped, and a drop
// leaves nothing behind to assert on — the setter is simply absent, which is
// indistinguishable from a struct that never declared the field.

package builder

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/node"

	"go.thesmos.sh/testkit"
)

// Reading the name off the reference rather than off the pointee is what made
// an embed by pointer vanish from every builder without a diagnostic, so both
// spellings are pinned here rather than only the one that was broken.
func TestEmbedName(t *testing.T) {
	t.Parallel()

	t.Run("takes the name of a type embedded by value", func(t *testing.T) {
		t.Parallel()
		name, pointer := embedName(storefixture.Named("Meta"))
		testkit.Equal(t, name, "Meta", "the reference names itself")
		testkit.False(t, pointer, "an embed by value is not a pointer")
	})

	t.Run("takes the name of a type embedded by pointer from its pointee", func(t *testing.T) {
		t.Parallel()
		// The reference carries no name of its own, so reading it yields the
		// empty string and the caller drops the field.
		name, pointer := embedName(storefixture.Pointer(storefixture.Named("Audit")))
		testkit.Equal(t, name, "Audit", "the pointee names the field")
		testkit.True(t, pointer, "an embed by pointer is reported as one")
	})

	t.Run("declines an embed with no recorded type", func(t *testing.T) {
		t.Parallel()
		name, _ := embedName(nil)
		testkit.Equal(t, name, "", "an embed with no type contributes no field")
	})

	t.Run("declines a pointer with no recorded pointee", func(t *testing.T) {
		t.Parallel()
		// Composing a setter from a name that is not there would emit a method
		// on a type the file never declares.
		name, _ := embedName(&node.TypeRef{TypeKind: node.TypeRefPointer})
		testkit.Equal(t, name, "", "a pointer to nothing contributes no field")
	})
}
