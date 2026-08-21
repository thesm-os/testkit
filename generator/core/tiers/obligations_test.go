// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/core/tiers"
)

// Every obligation belongs to exactly one tier, and the assignment is
// fixed by what the obligation needs.
//
// The suite half is the closed list: a claim needing anything not on it
// is somebody else's. Getting that wrong in the permissive direction is
// the vacuity both ADRs exist to prevent — a check for `cas` written
// where no stale version can be produced passes against every
// implementation, including a broken one.
func TestEveryObligationHasATier(t *testing.T) {
	t.Parallel()

	suiteOwned := 0
	for _, ob := range tiers.Obligations() {
		tier, known := ob.Tier()
		testkit.True(t, known, "obligation "+string(ob)+" is classified")
		if tier == tiers.TierSuite {
			suiteOwned++
		}
	}

	testkit.Len(t, tiers.Obligations(), 13, "the register's thirteen kinds")
	testkit.Equal(t, suiteOwned, 6,
		"the suite tier's vocabulary is six kinds and adding a seventh is a design event")
}

// An unclassified obligation reports the model tier, not the suite's.
//
// The fallback direction is the decision: reporting the suite tier for
// something nobody classified would let a claim be emitted where it
// cannot be stated. The caller is told it was a fallback so a census can
// tell a decision from a default.
func TestAnUnknownObligationFallsToTheTierWithMachinery(t *testing.T) {
	t.Parallel()

	tier, known := tiers.Obligation("something nobody wrote down").Tier()
	testkit.False(t, known, "it is not in the vocabulary")
	testkit.Equal(t, tier, tiers.TierModel, "and the safe default needs machinery to discharge")
}

// A law-backed classification has a model-tier obligation whether or not
// the suite tier also covers part of it.
func TestObligationsForReadsTheLawColumn(t *testing.T) {
	t.Parallel()

	t.Run("a classification with laws owes elsewhere", func(t *testing.T) {
		t.Parallel()
		// idempotent is the case ADR-0028 was written about: a suite row
		// for the repeat, two laws for the sequences.
		got := tiers.ObligationsFor("idempotent")
		testkit.Len(t, got, 1, "one obligation named")
		testkit.Equal(t, got[0], tiers.ObUniversal,
			"the true weaker statement, until the catalogue records which of the six")
	})

	t.Run("a classification with none owes nothing elsewhere", func(t *testing.T) {
		t.Parallel()
		testkit.Len(t, tiers.ObligationsFor("sideeffect"), 0,
			"a note pointing at a tier that checks nothing is worse than silence")
		testkit.Len(t, tiers.ObligationsFor("not a classification"), 0,
			"and an unknown word earns no note either")
	})
}
