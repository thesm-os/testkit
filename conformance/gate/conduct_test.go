// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import (
	"reflect"
	"slices"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/gate"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/engine/model/law"
	"go.thesmos.sh/testkit/generator/core/tiers"
)

// TestEveryLawHasAConduct holds the census total over the vocabulary, both
// directions.
//
// The classification is manual — nothing mechanical sees a mutation through a
// closure — so totality is the one property a test can add: a law landing in
// the catalogue without a verdict fails here by name, before anything decides
// it is safe to bind.
func TestEveryLawHasAConduct(t *testing.T) {
	t.Parallel()

	declared := lawid.All()
	for _, id := range declared {
		c, classified := gate.LawConduct[id]
		testkit.True(t, classified, id+" carries a conduct verdict")
		if classified {
			testkit.True(
				t,
				c.Sound() || c == gate.ConductNeedsMirror || c == gate.ConductNeedsIsolation,
				id+"'s verdict is from the vocabulary",
			)
		}
	}
	for id := range gate.LawConduct {
		testkit.True(t, slices.Contains(declared, id),
			id+" is an identifier the vocabulary declares")
	}
}

// TestNoUnsoundLawIsBindable is the census's teeth: the generator binds only
// what the instantiation column covers, so an unsound law must not be in it.
//
// This is what turns the audit from a survey into a gate. A law that mutates
// the pair without mirroring is sound alone and unsound interleaved, and the
// difference is invisible in its own tests — the corpus found the first one
// only because a fixture armed it. Every later one is refused here instead.
func TestNoUnsoundLawIsBindable(t *testing.T) {
	t.Parallel()

	for _, id := range tiers.Bound() {
		c := gate.LawConduct[id]
		testkit.True(t, c.Sound(),
			id+" is bindable ("+string(c)+"); an unsound law must not carry an instantiation row")
	}
}

// TestConductVocabularyIsClosed pins Sound's arms to the six spellings, so a
// verdict typo'd into the census reads as unsound rather than as a seventh
// conduct nothing defined.
func TestConductVocabularyIsClosed(t *testing.T) {
	t.Parallel()

	for _, c := range []gate.Conduct{
		gate.ConductObservational, gate.ConductMirrored,
		gate.ConductSelfCleaning, gate.ConductIsolated,
	} {
		testkit.True(t, c.Sound(), string(c)+" keeps the pair synchronized")
	}
	testkit.False(t, gate.ConductNeedsMirror.Sound(), "needs-mirror does not")
	testkit.False(t, gate.ConductNeedsIsolation.Sound(), "needs-isolation does not")
	testkit.False(t, gate.Conduct("nonesuch").Sound(), "and an unknown spelling is unsound")
}

// TestIsolatedMarkerMatchesCensus holds the runner's dispatch to the census's
// verdict. The isolated conduct has two flavors: a law that corrupts whatever
// subjects it is handed carries the [law.Isolated] marker, and the runner
// routes it to a throwaway pair; a law that builds its own subjects through a
// Factory field self-isolates inside Check and needs no routing at all. So a
// marked law must be censused isolated, and an isolated verdict must be earned
// one way or the other — a marker the census missed corrupts the shared pair
// the runner still walks it on, and a verdict neither flavor backs quarantines
// a law the runner would have walked safely.
func TestIsolatedMarkerMatchesCensus(t *testing.T) {
	t.Parallel()

	marker := reflect.TypeFor[law.Isolated]()
	for id, typ := range gate.LawTypes {
		marked := typ.Implements(marker) || reflect.PointerTo(typ).Implements(marker)
		factory, selfIsolating := typ.FieldByName("Factory")
		selfIsolating = selfIsolating && factory.Type.Kind() == reflect.Func
		isolated := gate.LawConduct[id] == gate.ConductIsolated

		if marked {
			testkit.True(
				t,
				isolated,
				id+" carries the IsolatedLaw marker, so its verdict must be isolated",
			)
		}
		if isolated {
			testkit.True(
				t,
				marked || selfIsolating,
				id+" is censused isolated, so it must carry the marker or build its own subjects through a Factory field",
			)
		}
	}
}
