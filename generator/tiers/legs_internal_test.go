// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal like partners_internal_test.go: the census must iterate
// the own-leg table's own keys — reaching it only through LegOf would
// let a misspelled key sit unreachable forever, which is the exact
// defect the census exists to catch.
package tiers

import (
	"slices"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/engine/suite"
)

func TestLegCensus(t *testing.T) {
	t.Parallel()

	t.Run("every own-leg row names a registered law", func(t *testing.T) {
		t.Parallel()
		all := lawid.All()
		for law := range ownLegs() {
			testkit.True(t, slices.Contains(all, law),
				law+" is a live identifier, not a spelling the leg table invented")
		}
	})

	t.Run("the clocked family rides its own leg from the binding fact", func(t *testing.T) {
		t.Parallel()
		class, own := LegOf(lawid.TTLExpiry)
		testkit.True(t, own, "a clock-reading law cannot ride the shared sequences")
		testkit.Equal(t, class, suite.ClassClocked, "and reports under the clocked class")
	})

	t.Run("an unlisted law bundles by default", func(t *testing.T) {
		t.Parallel()
		_, own := LegOf(lawid.ReadAfterWrite)
		testkit.False(t, own, "an observational law rides the shared sequences")
	})

	t.Run("the total answer holds over the whole registry", func(t *testing.T) {
		t.Parallel()
		for _, law := range lawid.All() {
			class, own := LegOf(law)
			if own {
				testkit.NotEqual(t, class, suite.Class(""), law+"'s own leg names its class")
			}
		}
	})
}
