// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

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

	// Dir is the fixture's corpus-relative directory and IfaceName its bare
	// interface name — together what the unarmed-door census needs to find
	// the consumer tests and compose the option spellings they would call.
	Dir       string
	IfaceName string

	// Doors maps each guarded law to the config fields its registration
	// reads, and Clocked lists the laws armed only on the run's clock —
	// both invisible skips unless a consumer arms them or a register row
	// argues why not. Unarmed maps each law to the optional roles nothing
	// declared.
	Doors   map[string][]string
	Clocked []string
	Unarmed map[string][]string

	// SentinelStamped reports a declaration whose miss identity the derived
	// oracle routes; SentinelArmed that the sequences carry it. The census
	// holds the first to imply the second — a stamp that stops reaching the
	// sequences is a silent regression of the identity comparison.
	SentinelStamped bool
	SentinelArmed   bool
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
		e := Emitted{Fixture: b.IfaceName, IfaceName: b.IfaceName, Twin: b.Reference.Twin()}
		if iface, ok := origin.(*sdk.Interface); ok {
			e.Fixture = iface.Package + "." + iface.Name
			e.Dir = strings.TrimPrefix(iface.Package, "go.thesmos.sh/testkit/conformance/corpus/")
		}
		for _, l := range b.Laws {
			e.Laws = append(e.Laws, l.ID)
			if len(l.Supplied) > 0 {
				if e.Doors == nil {
					e.Doors = map[string][]string{}
				}
				e.Doors[l.ID] = append(e.Doors[l.ID], l.Supplied...)
			}
			if l.Clocked {
				e.Clocked = append(e.Clocked, l.ID)
			}
			if len(l.Unarmed) > 0 {
				if e.Unarmed == nil {
					e.Unarmed = map[string][]string{}
				}
				e.Unarmed[l.ID] = append(e.Unarmed[l.ID], l.Unarmed...)
			}
		}
		e.SentinelStamped = b.Reference.MissSym != nil && b.Reference.Derived()
		for _, a := range b.Actions {
			if a.Sentinel != nil {
				e.SentinelArmed = true
			}
		}
		sort.Strings(e.Laws)
		sort.Strings(e.Clocked)
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Fixture < out[j].Fixture })
	return out, nil
}
