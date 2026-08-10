// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator"
)

// The set is what a binary registers, and every structural fault in it is one
// the pipeline already rejects at Build: a duplicate plugin name, two plugins
// declaring one directive schema, two providers of one capability in a bucket,
// an emit major nothing can render, a malformed output table. Asserting those
// here by hand would be a second, weaker copy of a check eidos runs anyway —
// and the hand-written one this replaces asserted only that each role was
// filled at least once, which no real fault looks like.
//
// RunSetSuite fills the roles the set does not claim, so it reports what is
// wrong with the set rather than what the set is not, and runs the per-plugin
// conformance suite over each member on the way past.
func TestAllIsAWorkingPluginSet(t *testing.T) {
	t.Parallel()

	plugintest.RunSetSuite(t, generator.All()...)
}

// Every generator must appear in the full set. A generator registered in
// Generators but dropped from All builds and tests cleanly, and silently never
// runs.
func TestAllContainsEveryGenerator(t *testing.T) {
	t.Parallel()

	all := make(map[string]struct{})
	for _, p := range generator.All() {
		all[p.Name()] = struct{}{}
	}

	gens := generator.Generators()
	if len(gens) == 0 {
		t.Fatal("no generators are registered at all")
	}
	for _, g := range gens {
		if _, ok := all[g.Name()]; !ok {
			t.Errorf("generator %q is registered but absent from All", g.Name())
		}
	}
}

// The shape plugin is three annotators. Registering fewer is silent in a way
// no coverage gate can see — the classifier stamps either way — so the count
// is asserted here, where a companion dropped from the set is a failure rather
// than a class of enforcement that quietly stops running.
func TestAnnotatorsCarryEveryShapeCompanion(t *testing.T) {
	t.Parallel()

	want := generator.Annotator().Annotators()
	got := generator.Annotators()

	names := make(map[string]struct{}, len(got))
	for _, a := range got {
		names[a.Name()] = struct{}{}
	}
	for _, w := range want {
		if _, ok := names[w.Name()]; !ok {
			t.Errorf("annotator %q is a shape companion but absent from Annotators", w.Name())
		}
	}
}

// The annotator is configured here so the CLI and the conformance gate cannot
// disagree about which classifications exist. An unconfigured one stamps
// nothing, which would read as an empty corpus rather than as a wiring fault.
func TestAnnotatorIsConfigured(t *testing.T) {
	t.Parallel()

	// The role itself is enforced by the return type, so what is left to check
	// is that one was configured and that it satisfies the interface the set
	// registers it under.
	a := generator.Annotator()
	if a == nil {
		t.Fatal("no annotator is configured")
	}
	var _ sdk.Annotator = a
}
