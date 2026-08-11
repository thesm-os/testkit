// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/tiers"
)

// TestEveryDetectorDrivesAnAction holds the action table total over the live
// registry.
//
// A detector without a row is a method silently absent from every generated
// sequence — the run still passes, over sequences that never call it. eidos
// adding a detector must fail this test by name, in the same build that makes
// the classification stampable.
func TestEveryDetectorDrivesAnAction(t *testing.T) {
	t.Parallel()

	for _, d := range detectors.All() {
		ctor, ok := tiers.ActionFor(d.Name)
		testkit.True(t, ok, d.Name+" names an action constructor")
		testkit.True(t, ctor != "", d.Name+"'s constructor is not empty")
	}
}

// TestActionForDeclinesTheUnknown pins the miss arm: a name outside the
// vocabulary answers false rather than an empty constructor a template would
// render as a call to nothing.
func TestActionForDeclinesTheUnknown(t *testing.T) {
	t.Parallel()

	ctor, ok := tiers.ActionFor("not-a-shape")
	testkit.False(t, ok, "an unregistered shape names no constructor")
	testkit.Equal(t, ctor, "", "and returns nothing to render")
}

// TestMapStoreOpsAreDetectorShapes holds the oracle's delegation rows to the
// same vocabulary as the action rows.
//
// A row keyed on a name no detector stamps is delegation nothing can reach;
// the adapter renders every stamped method either through a row or inert, so
// an unreachable row is a modelled shape that silently became inert.
func TestMapStoreOpsAreDetectorShapes(t *testing.T) {
	t.Parallel()

	live := map[string]bool{}
	for _, d := range detectors.All() {
		live[d.Name] = true
	}
	for _, s := range tiers.MapStoreShapes() {
		op, ok := tiers.MapStoreOp(s)
		testkit.True(t, ok, s+" is a shape the oracle models")
		testkit.True(t, op != "", s+"'s delegation names a method")
		testkit.True(t, live[s], s+" is a shape the annotator stamps")
	}
}
