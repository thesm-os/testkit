// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator_test

import (
	"maps"
	"testing"
	"text/template"

	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/lang/golang"
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

// Every function a shipped template calls has to be one somebody in the run
// provides — the plugin's own funcmap, or the backend's reserved set.
//
// [plugintest.RunSuite] does not ask this. It parses each template with every
// unresolved name stubbed, deliberately, so that it judges syntax alone; a
// template calling a function nobody registers therefore parses, ships, and
// fails midway through Render in the consumer's build, naming the merged tree
// rather than the file.
//
// Asserted over the set rather than beside each plugin, because it is one
// question with one answer and five copies would be five things to remember
// when a sixth generator arrives.
func TestEveryTemplateFuncResolves(t *testing.T) {
	t.Parallel()

	for _, p := range generator.All() {
		t.Run(p.Name(), func(t *testing.T) {
			t.Parallel()
			plugintest.AssertTemplateFuncsResolve(t, p, reservedFuncs(), golang.Language)
		})
	}
}

// reservedFuncs is everything the Go backend brings to a template, assembled
// from the three places it is reachable from.
//
// The assertion parses each template with this map, and parsing resolves names
// without calling bodies — so a placeholder binds as well as the real function,
// and a variadic one binds against every call shape a template can write.
//
// Two of the three sources are authoritative. plugintest exports the canonical
// reserved names, and lang/golang exports the shared Go conventions the backend
// layers on top. [backendExtras] is the third and is hand-kept, because the
// backend keeps that category unexported — see its own docblock.
func reservedFuncs() template.FuncMap {
	out := template.FuncMap{}
	for _, name := range plugintest.ReservedTemplateFuncNames() {
		out[name] = placeholderFunc
	}
	for _, name := range backendExtras {
		out[name] = placeholderFunc
	}
	maps.Copy(out, golang.FuncMap())
	return out
}

// placeholderFunc stands in for a backend helper whose body this check never
// calls. Variadic and untyped so it binds against any call a template writes:
// what is under test is whether the *name* resolves.
func placeholderFunc(...any) any { return nil }

// backendExtras names the overrideable helpers the Go backend registers —
// naming, meta-read, string and debug — which every plugin template may call.
//
// Hand-kept, and the only hand-kept list here. `extrasFuncMap` is unexported
// and `plugintest.ReservedTemplateFuncNames` reports the canonical set only, so
// there is no accessor to read. The drift this admits is loud rather than
// silent: a helper eidos adds is one this list lacks, which fails the check for
// a template that is in fact correct — someone fixes it the same day. The
// opposite list, of names a template calls, is the one that drifts silently,
// and that half is read from the parser.
//
// Reported upstream; delete this and pass the accessor when one ships.
//
//nolint:gochecknoglobals // immutable lookup table.
var backendExtras = []string{
	// Naming.
	"pascal", "camel", "snake", "screaming", "exported",
	// Meta read.
	"meta", "metaBool", "metaStr", "hasMeta", "metaEq",
	// String.
	"join", "title", "upper", "lower", "trim", "split", "default", "coalesce",
	// Debug.
	"origin", "explain",
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
