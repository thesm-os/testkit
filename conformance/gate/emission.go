// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

	backendgolang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/frontend/golang"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sdk"
	"go.thesmos.sh/eidos/sink"

	"go.thesmos.sh/testkit/core/brand"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/model"
)

// Emitted is what the model tier actually asserted for one interface: the
// law identifiers bound into its generated registry, and the reference kind
// the sequences ran against.
type Emitted struct {
	// Fixture is the interface's package path — the corpus address a red
	// gate prints.
	Fixture string

	// Laws are the bound identifiers; Twin reports the reference floor.
	Laws []string
	Twin bool
}

// Emission runs the real generators over the corpus in memory and reports
// what each armed interface bound — the assertion half of the gate.
//
// [Annotate] measures that a classification is stamped somewhere;
// [Coverage.Elsewhere] measures that a law for it exists in the catalogue.
// Neither measures that the stamp bought an assertion, which is the gap the
// generated-suite audit proved by deleting a fixture's whole claim and
// watching the corpus stay green. This is the measurement that closes it:
// the same pipeline the CLI runs, the same plugin set, the queued
// [model.Bindings] read back before any file is rendered.
//
// # Hazards
//
// The run executes every generator, not just the model tier: the model
// plugin reads the suite's projection from the emit queue, and a subset
// would measure a pipeline production never runs. Like [Annotate], the run
// is entirely in memory.
func Emission(ctx context.Context, root string, patterns ...string) ([]Emitted, error) {
	scoped := make([]string, len(patterns))
	for i, p := range patterns {
		scoped[i] = filepath.Join(root, p)
	}

	builder := pipeline.New().
		WithBrand(brand.Name).
		WithDirectivePrefix(brand.DirectivePrefix).
		WithSourceRoot(root).
		WithFrontend(golang.New())
	for _, a := range generator.Annotators() {
		builder = builder.WithAnnotator(a)
	}
	for _, g := range generator.Generators() {
		// Every testkit generator implements the role; the registry's type
		// is the plugin universe's, so the assertion narrows it back.
		if gen, ok := g.(plugin.Generator); ok {
			builder = builder.WithGenerator(gen)
		}
	}

	pipe, err := builder.
		WithBackend(backendgolang.New()).
		WithSink(sink.NewMemory()).
		Build()
	if err != nil {
		return nil, fmt.Errorf("gate: build emission pipeline: %w", err)
	}
	if err := pipe.Run(ctx, scoped...); err != nil {
		return nil, fmt.Errorf("gate: run emission pipeline: %w", err)
	}

	out := make([]Emitted, 0, 128)
	for origin, b := range sdk.PendingByOrigin[*model.Bindings](pipe.Store().Emit()) {
		e := Emitted{Fixture: b.IfaceName, Twin: b.Reference.Twin()}
		if iface, ok := origin.(*sdk.Interface); ok {
			e.Fixture = iface.Package + "." + iface.Name
		}
		for _, l := range b.Laws {
			e.Laws = append(e.Laws, l.ID)
		}
		sort.Strings(e.Laws)
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Fixture < out[j].Fixture })
	return out, nil
}
