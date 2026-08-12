// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import (
	"reflect"
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/gate"
	"go.thesmos.sh/testkit/engine/model/ref"
	"go.thesmos.sh/testkit/generator/tiers"
)

// TestEveryActionRowNamesAShippedConstructor closes the action table's engine
// side.
//
// The tiers side already holds every detector to a row; this holds every row's
// answer to a function `engine/model/action` exports. Between the two, the
// table cannot name a constructor that does not ship and cannot miss a shape
// that does — the drift the last hand-maintained mapping was retired for.
func TestEveryActionRowNamesAShippedConstructor(t *testing.T) {
	t.Parallel()

	for _, d := range detectors.All() {
		ctor, ok := tiers.ActionFor(d.Name)
		testkit.True(t, ok, d.Name+" has a row")

		fn, shipped := gate.ActionCtors[ctor]
		testkit.True(t, shipped, d.Name+"'s constructor "+ctor+" is in the census")
		if !shipped {
			continue
		}
		testkit.Equal(t, reflect.TypeOf(fn).Kind(), reflect.Func,
			ctor+" is a function")
	}
}

// TestCensusCarriesNoRetiredConstructor is the other direction: an entry here
// that no detector's row reaches is a constructor the census claims is in use
// and is not — the stale-excuse problem, one table over.
func TestCensusCarriesNoRetiredConstructor(t *testing.T) {
	t.Parallel()

	reached := map[string]bool{}
	for _, d := range detectors.All() {
		if ctor, ok := tiers.ActionFor(d.Name); ok {
			reached[ctor] = true
		}
	}
	for name := range gate.ActionCtors {
		testkit.True(t, reached[name], name+" is reached by some detector's row")
	}
}

// TestEveryMapStoreOpIsAMethod holds the oracle delegation rows to the shipped
// oracle.
//
// The generated adapter forwards a shape's parameters in order and changes
// only the name, which is sound exactly while the named method exists with the
// shape's own signature. Existence is checked here; the signature agreement is
// checked where it cannot lie — the corpus, which compiles the adapter.
func TestEveryMapStoreOpIsAMethod(t *testing.T) {
	t.Parallel()

	store := reflect.TypeFor[ref.MapStore[string, string]]()
	for _, s := range tiers.MapStoreShapes() {
		op, _ := tiers.MapStoreOp(s)
		testkit.True(t, gate.HasMethod(store, op),
			s+" delegates to MapStore."+op+", which exists")
	}

	keyed := reflect.TypeFor[ref.KeyedStore[string, string]]()
	for _, s := range tiers.KeyedStoreShapes() {
		op, _ := tiers.KeyedStoreOp(s)
		testkit.True(t, gate.HasMethod(keyed, op),
			s+" delegates to KeyedStore."+op+", which exists")
	}
	op, assigned := tiers.KeyedStoreMixinOp("deleteremoves")
	testkit.True(t, assigned && gate.HasMethod(keyed, op),
		"the delete assignment names a method the keyed oracle declares")

	// Both collection forms carry the same surface, so one op table serves
	// the log and the set the dedupe claim refines it into.
	log := reflect.TypeFor[ref.Collection[string]]()
	set := reflect.TypeFor[ref.SetCollection[string]]()
	for _, s := range []string{"writer", tiers.ShapeCollector} {
		op, ok := tiers.CollectionOp(s)
		testkit.True(t, ok, s+" is a shape the collection models")
		testkit.True(t, gate.HasMethod(log, op) && gate.HasMethod(set, op),
			s+" delegates to "+op+", which both forms declare")
	}
	testkit.True(t, tiers.CollectionDedupes("noduplicates"),
		"the dedupe refinement follows the claim")
	testkit.False(t, tiers.CollectionDedupes("permutation"),
		"and only the claim")
}
