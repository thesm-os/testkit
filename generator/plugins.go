// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator

import (
	backendgolang "go.thesmos.sh/eidos/backend/golang"
	frontendgolang "go.thesmos.sh/eidos/frontend/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/full"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/builder"
	"go.thesmos.sh/testkit/generator/defaults"
	"go.thesmos.sh/testkit/generator/enum"
	"go.thesmos.sh/testkit/generator/fault"
	"go.thesmos.sh/testkit/generator/model"
	"go.thesmos.sh/testkit/generator/sentinel"
	"go.thesmos.sh/testkit/generator/stub"
	"go.thesmos.sh/testkit/generator/suite"
)

// Annotator returns the shape annotator configured with every classification
// eidos registers.
//
// The concrete type is returned rather than [sdk.Annotator] because the
// classifier is one of three companions built from the same registrations, and
// the method that returns all three is not on the interface. [Annotators]
// registers them.
//
// It is defined here rather than at each call site because two things
// configure it — the CLI that generates, and the conformance gate that
// measures what the corpus stamps — and those must agree. A gate measuring a
// narrower set than the CLI applies would report coverage the generated output
// does not have; a wider one would demand fixtures for classifications no
// build can produce.
//
// The full registry is taken deliberately, and taken from eidos rather than
// assembled here. testkit does not curate the classification vocabulary
// (docs/adr/0004), so a classification added upstream is available the moment
// the dependency moves — and the gate starts asking for a fixture in the same
// build. Naming the three axes here instead would put a list in testkit that
// has to be remembered when a fourth arrives.
func Annotator() *shape.Plugin { return full.New() }

// Annotators returns every annotator a run registers: eidos's shape
// classifier, its contract resolver, and testkit's own.
//
// All are needed wherever source is read, not only where code is generated.
// An annotator owns its directive schema, so a pipeline missing one rejects
// the directives it declares as unknown — which is why the conformance gate
// takes this set rather than [Annotator] alone.
//
// The shape plugin is three annotators, not one, and the set comes from eidos
// rather than being composed here. The classifier stamps; the resolver, one
// bucket later, rewrites a contract's partner reference into a qualified name
// and back-stamps the membership onto that partner; the validator, one bucket
// later again, enforces every [shape.Contract.Required] declaration and runs
// each contract's and mixin's Validate hook.
//
// Each omission is silent in its own way. Without the resolver a partner
// reference stays as the author wrote it and no generator can turn it into a
// call. Without the validator every Required declaration and every Validate
// hook in the catalogue is dead — a contract missing a mandatory partner, or
// one whose members contradict each other, passes. Neither shows up in a
// coverage gate: the declaring side stamps the classification either way.
func Annotators() []sdk.Annotator {
	return append(Annotator().Annotators(), defaults.New(), fault.New())
}

// Generators returns the generator plugins this build carries.
//
// Order is alphabetical and carries no meaning: eidos resolves execution order
// from each plugin's declared priority and capabilities, so a generator that
// must follow another says so through Requires rather than by being listed
// later.
func Generators() []sdk.Plugin {
	return []sdk.Plugin{
		builder.New(),
		enum.New(),
		model.New(),
		sentinel.New(),
		stub.New(),
		suite.New(),
	}
}

// All returns the complete plugin universe a testkit binary registers:
// frontend, annotator, generators, and backend.
//
// The frontend and backend are eidos's Go implementations. testkit supplies
// neither — parsing Go and rendering Go are the substrate's job, and the
// boundary is what docs/adr/0003 exists to hold.
func All() []sdk.Plugin {
	annotators, generators := Annotators(), Generators()

	out := make([]sdk.Plugin, 0, len(annotators)+len(generators)+2)
	out = append(out, frontendgolang.New())
	for _, a := range annotators {
		out = append(out, a)
	}
	out = append(out, generators...)
	return append(out, backendgolang.New())
}
