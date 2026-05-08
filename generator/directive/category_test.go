// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directive_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
)

func TestCategory(t *testing.T) {
	t.Parallel()

	t.Run("String returns canonical category name", func(t *testing.T) {
		t.Parallel()
		cases := map[directive.Category]string{
			directive.SignatureHint:       "SignatureHint",
			directive.ContractTier:        "ContractTier",
			directive.Mixin:               "Mixin",
			directive.Enrichment:          "Enrichment",
			directive.Documentation:       "Documentation",
			directive.CategoryUnspecified: "Unspecified",
		}
		for c, want := range cases {
			testkit.Equal(t, c.String(), want, "category String")
		}
	})

	t.Run("recognised categories are distinct values", func(t *testing.T) {
		t.Parallel()
		seen := map[directive.Category]bool{}
		for _, c := range []directive.Category{
			directive.SignatureHint, directive.ContractTier,
			directive.Mixin, directive.Enrichment, directive.Documentation,
		} {
			testkit.False(t, seen[c], "category value collision: "+c.String())
			seen[c] = true
		}
	})
}
