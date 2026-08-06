// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator

import (
	backendgolang "go.thesmos.sh/eidos/backend/golang"
	frontendgolang "go.thesmos.sh/eidos/frontend/golang"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins"

	"go.thesmos.sh/testkit/generator/builder"
	"go.thesmos.sh/testkit/generator/enum"
	"go.thesmos.sh/testkit/generator/internal/defaults"
	"go.thesmos.sh/testkit/generator/internal/fault"
	"go.thesmos.sh/testkit/generator/sentinel"
	"go.thesmos.sh/testkit/generator/stub"
)

// Annotator returns the shape annotator configured with every classification
// eidos registers.
//
// The concrete type is returned rather than [plugin.Annotator] because the
// classifier is only half of the shape plugin: [shape.Plugin.Resolver] builds
// its refinement-bucket companion from the same registrations, and that method
// is not on the interface. [Annotators] registers both.
//
// It is defined here rather than at each call site because two things
// configure it — the CLI that generates, and the conformance gate that
// measures what the corpus stamps — and those must agree. A gate measuring a
// narrower set than the CLI applies would report coverage the generated output
// does not have; a wider one would demand fixtures for classifications no
// build can produce.
//
// The full registry is taken deliberately. testkit does not curate eidos's
// classification vocabulary (docs/adr/0004), so a classification added
// upstream is available the moment the dependency moves — and the gate starts
// asking for a fixture in the same build.
func Annotator() *shape.Plugin {
	return shape.New().
		Detectors(detectors.All()...).
		Contracts(contracts.All()...).
		Mixins(mixins.All()...)
}

// Annotators returns every annotator a run registers: eidos's shape
// classifier, its contract resolver, and testkit's own.
//
// All are needed wherever source is read, not only where code is generated.
// An annotator owns its directive schema, so a pipeline missing one rejects
// the directives it declares as unknown — which is why the conformance gate
// takes this set rather than [Annotator] alone.
//
// The resolver is the classifier's companion and shares its registrations. It
// runs one priority bucket later and is what turns a contract's partner
// reference from the name the author wrote into a qualified one, back-stamps
// the membership onto that partner so both sides of a protocol carry it, and
// reports a role or a partner that resolves to nothing. Registering the
// classifier alone leaves all three undone, and silently: the declaring side
// still stamps the contract, so a coverage gate counting stamped
// classifications sees nothing wrong.
func Annotators() []plugin.Annotator {
	classifier := Annotator()
	return []plugin.Annotator{
		classifier,
		classifier.Resolver(),
		defaults.New(),
		fault.New(),
	}
}

// Generators returns the generator plugins this build carries.
//
// Order is alphabetical and carries no meaning: eidos resolves execution order
// from each plugin's declared priority and capabilities, so a generator that
// must follow another says so through Requires rather than by being listed
// later.
func Generators() []plugin.Plugin {
	return []plugin.Plugin{
		builder.New(),
		enum.New(),
		sentinel.New(),
		stub.New(),
	}
}

// All returns the complete plugin universe a testkit binary registers:
// frontend, annotator, generators, and backend.
//
// The frontend and backend are eidos's Go implementations. testkit supplies
// neither — parsing Go and rendering Go are the substrate's job, and the
// boundary is what docs/adr/0003 exists to hold.
func All() []plugin.Plugin {
	annotators := Annotators()
	out := make([]plugin.Plugin, 0, len(Generators())+len(annotators)+2)
	out = append(out, frontendgolang.New())
	for _, a := range annotators {
		out = append(out, a)
	}
	out = append(out, Generators()...)
	return append(out, backendgolang.New())
}
