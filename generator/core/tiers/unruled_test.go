// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/core/tiers"
)

// Every shipped law is selected by a rule or argued about by name.
//
// A law nothing selects is the quietest failure this catalogue has. The
// engine implements it, lawid declares it, the conformance census counts
// it toward the total — and no generated file binds it, so it is
// evidence nobody ever collects and nothing says so. Two are argued
// today; a third appearing means a law was added and its rule was not.
func TestEveryLawIsRuledOrArgued(t *testing.T) {
	t.Parallel()

	testkit.Equal(t, tiers.Unaccounted(), []string(nil),
		"a law with no rule and no row is a property the engine implements and "+
			"no consumer will ever run; give it a rule, or a row saying why it cannot have one")
}

// A row that outlived its gap is removed, or it goes on asserting
// something false.
//
// The direction a register loses without noticing: somebody adds the
// rule, the row stays, and the next reader believes it. The census would
// then hide a law that IS bound behind an argument for why it is not.
func TestNoUnruledRowIsStale(t *testing.T) {
	t.Parallel()

	testkit.Equal(t, tiers.StaleUnruledRows(), []string(nil),
		"these laws now have rules, so the rows arguing that nothing selects them are wrong")
}

// Each row argues its case in a sentence somebody can disagree with.
func TestEveryUnruledRowArguesItsCase(t *testing.T) {
	t.Parallel()

	for law, why := range tiers.UnruledLaws {
		testkit.True(t, len(why) > 40,
			"the row for "+law+" states a reason rather than a label")
	}
}

// The register names laws, not spellings of them.
func TestUnruledRowsNameDeclaredLaws(t *testing.T) {
	t.Parallel()

	declared := lawid.All()
	for law := range tiers.UnruledLaws {
		testkit.Contains(t, declared, law,
			"the register rows "+law+", which core/lawid does not declare")
	}
}

// Ruled and the register partition what lawid declares.
func TestRuledAndUnruledPartitionTheLaws(t *testing.T) {
	t.Parallel()

	testkit.Equal(t, len(tiers.Ruled())+len(tiers.UnruledLaws), len(lawid.All()),
		"every declared law is on exactly one side, so neither list can drift alone")
}
