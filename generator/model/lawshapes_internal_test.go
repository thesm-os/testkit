// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"slices"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
)

// The table transcribes each shipped law struct's role fields, and its
// keys have to be laws the engine actually declares.
//
// A key nothing declares is a row that never fires: the binding walks
// the laws a method owes, finds no entry, and refuses by name — which
// reads as "this law has no shape yet" when the truth is that the row is
// there under a misspelling.
func TestRoleShapesKeyOnDeclaredLaws(t *testing.T) {
	t.Parallel()

	declared := lawid.All()
	for law := range lawRoleShapes {
		testkit.True(t, slices.Contains(declared, law),
			"lawRoleShapes rows "+law+", which core/lawid does not declare")
	}
}

// Every role a row binds resolves to a closure template.
//
// A rowed law's role field missing a template is a build refusal by
// name, never a wrong guess — but only if the shape it names exists.
func TestRoleShapesResolveToTemplates(t *testing.T) {
	t.Parallel()

	for law, roles := range lawRoleShapes {
		testkit.NotEqual(t, len(roles), 0, "law "+law+" is rowed with no role field")
		for role, shape := range roles {
			testkit.True(t, templateExists(t, shape),
				"law "+law+" binds "+role+" at shape "+string(shape)+", which has no template")
		}
	}
}

// A role spelling is a constant, because the same word names the same
// concept across rows and a literal repeated per row is a rename that
// changes some of them.
func TestRoleSpellingsAreShared(t *testing.T) {
	t.Parallel()

	// Read is the field spelling most rows carry, and the one a typo
	// would be least visible in.
	testkit.Equal(t, lawRoleShapes[lawid.ReadAfterWrite][fRead], shapeKeyedRead,
		"a keyed read on a store is the keyed-read closure")
	testkit.Equal(t, lawRoleShapes[lawid.Cacheable][fRead], shapeKeyedRead,
		"and the same field on another law is the same closure, by the same constant")
}
