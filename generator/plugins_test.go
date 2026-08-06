// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugin"

	"go.thesmos.sh/testkit/generator"
)

// The binary is useless without all four roles: a missing frontend parses
// nothing, a missing backend renders nothing, and either surfaces as a
// pipeline build error at run time rather than here.
func TestAllCarriesEveryRole(t *testing.T) {
	t.Parallel()

	var frontends, annotators, generators, backends int
	for _, p := range generator.All() {
		if _, ok := p.(plugin.Frontend); ok {
			frontends++
		}
		if _, ok := p.(plugin.Annotator); ok {
			annotators++
		}
		if _, ok := p.(plugin.Generator); ok {
			generators++
		}
		if _, ok := p.(plugin.Backend); ok {
			backends++
		}
	}

	for _, c := range []struct {
		role string
		n    int
	}{
		{"frontend", frontends},
		{"annotator", annotators},
		{"generator", generators},
		{"backend", backends},
	} {
		if c.n == 0 {
			t.Errorf("the plugin set registers no %s", c.role)
		}
	}
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

// The annotator is configured here so the CLI and the conformance gate cannot
// disagree about which classifications exist. An unconfigured one stamps
// nothing, which would read as an empty corpus rather than as a wiring fault.
func TestAnnotatorIsConfigured(t *testing.T) {
	t.Parallel()

	// The role itself is enforced by the return type, so the only thing left
	// to check at runtime is that one was actually configured.
	if generator.Annotator() == nil {
		t.Fatal("no annotator is configured")
	}
}
