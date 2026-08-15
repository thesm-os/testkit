// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"slices"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/model"
)

// Every wear says what it does wrong.
//
// Totality over the wardrobe, and the half that decides whether a kill counts.
// A wear the table does not name produces no class, intersects no law, and
// silently stops counting — a defect that used to prove something and now
// proves nothing, with a green row either way.
func TestEveryWearDeclaresADefectClass(t *testing.T) {
	t.Parallel()

	testkit.Len(t, model.Wardrobe(), 20, "the wardrobe is twenty wears and each is classified")
	for _, kind := range model.Wardrobe() {
		testkit.True(t, len(model.ClassesOf(kind)) > 0, kind+" says what it does wrong")
	}
}

// A wear's declared class is one the vocabulary has.
func TestEveryWearClassIsInTheVocabulary(t *testing.T) {
	t.Parallel()

	for _, kind := range model.Wardrobe() {
		for _, c := range model.ClassesOf(kind) {
			testkit.True(t, slices.Contains(lawid.Classes(), c),
				kind+" produces "+string(c)+", which is in the vocabulary")
		}
	}
}

// Three classes the wardrobe cannot produce, and no more.
//
// The join the whole axis turns on, held from this side as a count rather than
// as an absence: a class the laws claim and the wardrobe cannot produce is a
// promise nothing can test, and every law claiming one is unprovable by
// construction. Which laws those are, and what the missing wear would have to
// do, is `gate.UnprovableLaws` — this holds the number so a third gap cannot
// open without somebody deciding it should.
func TestThreeClassesHaveNoWear(t *testing.T) {
	t.Parallel()

	produced := map[lawid.DefectClass]bool{}
	for _, kind := range model.Wardrobe() {
		for _, c := range model.ClassesOf(kind) {
			produced[c] = true
		}
	}
	var gaps []string
	for _, c := range lawid.Classes() {
		if !produced[c] {
			gaps = append(gaps, string(c))
		}
	}
	testkit.Equal(t, strings.Join(gaps, ", "), "atomicity, resource, permissive",
		"a partial effect, a retained resource and a missing refusal are what no "+
			"dressing over a signature can produce — the first two add nothing this "+
			"wardrobe has words for, and the third would have to *remove* a check "+
			"where every wear here adds wrongness")
}

// Proves is an intersection, and answers no where there is none.
//
// The criterion itself, pinned on the two pairings the finding named. A Put
// that does nothing breaks every claim about what a Put leaves behind, which is
// why the old criterion counted it for `AUTO-CAS-ATOMIC-ONE-WINNER` — a law
// about two writers contending, proved by a subject with no writer at all.
func TestProvesRequiresASharedClass(t *testing.T) {
	t.Parallel()

	testkit.True(t, model.Proves("wane", lawid.MonotonicNonDecreasing),
		"a counter that runs backwards is what a non-decreasing claim is named for")
	testkit.False(t, model.Proves("inert", lawid.CASAtomicOneWinner),
		"a Put that does nothing is not two winners, which is the defect CAS names")
	// The finding's other example, and the classification refuses it. A
	// point-in-time claim is that a snapshot keeps answering as of its
	// instant, so a read that flickers breaks it as itself — the kill was
	// specific and the finding read it as generic. One of the two examples
	// survived the taxonomy and one did not, which is what a taxonomy is for.
	testkit.True(t, model.Proves("flap", lawid.PointInTime),
		"an unstable read is exactly how a point-in-time snapshot fails")
	testkit.False(t, model.Proves("not-a-wear", lawid.MonotonicNonDecreasing),
		"an unclassified wear proves nothing rather than everything")
}

// Every registered unreached law names a mechanism.
//
// The bar the other registers hold their reasons to, and it bites harder here:
// this register's rows are the only place a reader learns *why* a wear of the
// right class does not arrive, and "it does not fire" is the finding restated
// rather than an answer.
func TestEveryUnreachedRowNamesAMechanism(t *testing.T) {
	t.Parallel()

	for _, id := range model.UnreachedLaws() {
		testkit.True(t, len(model.Unreached(id)) > 70,
			id+"'s row says how the dressing fails to arrive, not that it does")
	}
}

// A law is registered as unreached only where a wear of its class exists.
//
// The two registers answer different questions and a law in both would mean
// nobody decided which. `gate.UnprovableLaws` holds the laws whose class the
// wardrobe cannot produce at all; this one holds laws where it can and the
// dressing still does not arrive. A row here for a law with no proving wear is
// an excuse standing in front of a gap.
func TestUnreachedOnlyCoversProvableLaws(t *testing.T) {
	t.Parallel()

	for _, id := range model.UnreachedLaws() {
		provable := false
		for _, kind := range model.Wardrobe() {
			if model.Proves(kind, id) {
				provable = true
			}
		}
		testkit.True(t, provable,
			id+" has a wear of its own class; a law that has none belongs in UnprovableLaws")
	}
}
