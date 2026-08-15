// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/gate"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/model"
)

// Every law is provable by a wear of its own class, or argued.
//
// What the defect-class axis buys. The saturation prover asks whether some
// defect made a law fail, and every law passes that — `AUTO-POINT-IN-TIME` on a
// read-purity flap, `AUTO-CAS-ATOMIC-ONE-WINNER` on a Put that did nothing. A
// law proved only by defects its name never mentions is a law whose name is a
// guess, and this is the census that says which ones those are.
func TestEveryLawIsProvableOrArgued(t *testing.T) {
	t.Parallel()

	unprovable := gate.Unprovable()
	testkit.True(t, len(unprovable) == 0,
		"every law has a wear of its own defect class, or a row in UnprovableLaws — "+
			"unregistered: "+strings.Join(unprovable, ", "))
}

// The register only shrinks.
//
// A row for a law some wear now proves is a stale excuse, and this register's
// rows are the wardrobe's backlog — leaving one after the wear lands would hide
// the very progress it was written to track.
func TestNoUnprovableRowIsStale(t *testing.T) {
	t.Parallel()

	stale := gate.ArguedButProvable()
	testkit.True(t, len(stale) == 0,
		"delete the rows for laws a wear now proves: "+strings.Join(stale, ", "))
}

// Every row says what the missing wear would have to do.
//
// The bar the other four registers hold their reasons to, and the one that
// makes this register a backlog rather than a list. "No wear covers it" is the
// finding restated; what a reader needs is why no *signature-derived* wear can,
// which is what says whether the fix is a wear or a different mechanism.
func TestEveryUnprovableRowNamesTheMissingWear(t *testing.T) {
	t.Parallel()

	for id, why := range gate.UnprovableLaws {
		testkit.True(t, len(why) > 60, id+"'s row says what the missing wear would have to do")
	}
}

// The two tables meet.
//
// A class the laws claim and the wardrobe cannot produce is the whole content
// of this register, so the join is what the register is a census of. Held here
// rather than in either table, because neither owns the pairing and a check
// living in one would be a check the other could break.
func TestEveryClaimedClassIsProducedOrItsLawsAreArgued(t *testing.T) {
	t.Parallel()

	produced := map[lawid.DefectClass]bool{}
	for _, kind := range model.Wardrobe() {
		for _, c := range model.ClassesOf(kind) {
			produced[c] = true
		}
	}

	var gaps []string
	for _, c := range lawid.Classes() {
		if produced[c] {
			continue
		}
		gaps = append(gaps, string(c))
	}
	// Two, and named: every law claiming either is in the register above, so
	// the count here and the rows there move together.
	testkit.Equal(t, strings.Join(gaps, ", "), "atomicity, resource, permissive",
		"the wardrobe cannot produce these three classes, and the laws claiming "+
			"them are the whole of UnprovableLaws")
}

// The census selects, and both arms are reachable only from here.
//
// The corpus answer to both questions is "none" — that is the state the gate
// exists to hold — so the arms that find something never run against it. A
// selector whose reporting half is never exercised is a gate that has only ever
// been seen agreeing, which is the shape of check this whole programme distrusts.
func TestUnprovableSelectsBothWays(t *testing.T) {
	t.Parallel()

	t.Run("a law with no class-matching wear and no row", func(t *testing.T) {
		t.Parallel()
		// AUTO-ATOMIC-WRITE is unprovable and registered; dropping its row is
		// what the unregistered arm looks like.
		testkit.True(t, gate.Reports(lawid.AtomicWrite, map[string]string{}),
			"an unprovable law nothing argues is reported")
		testkit.False(t, gate.Reports(lawid.AtomicWrite, gate.UnprovableLaws),
			"and the row is what silences it")
	})

	t.Run("a row for a law some wear proves", func(t *testing.T) {
		t.Parallel()
		// AUTO-MONOTONIC-NON-DECREASING is proved by `wane`, so a row for it
		// is the stale excuse the other arm reports.
		stale := gate.Stale(map[string]string{lawid.MonotonicNonDecreasing: "an excuse"})
		testkit.Equal(t, strings.Join(stale, ", "), lawid.MonotonicNonDecreasing,
			"a row for a provable law is named for deletion")
		testkit.Len(t, gate.Stale(gate.UnprovableLaws), 0, "and the real register has none")
	})
}
