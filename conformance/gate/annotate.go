// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

import (
	"context"
	"fmt"
	"path/filepath"

	backendgolang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/frontend/golang"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sink"

	"go.thesmos.sh/testkit/core/brand"
	"go.thesmos.sh/testkit/generator"
)

// Annotate runs eidos's frontend and shape annotator over the corpus and
// returns the classifications actually stamped, keyed by axis.
//
// This is the measurement half of the gate. [Compare] diffs the result against
// the registries; the point of producing it by running the real annotator —
// rather than by reading directory names or a manifest — is that a directive
// the annotator declines to read stamps nothing and shows up as a gap. A name
// check would pass, because the directory is named correctly and only the
// directive inside it is wrong.
//
// The run is entirely in memory. A backend is registered because the pipeline
// requires one, but with no generators there is no emit graph for it to
// render, and the sink is [sink.NewMemory] — so the full sequence executes and
// nothing reaches disk.
//
// # Hazards
//
// The annotator is upstream code, so a classification bug there surfaces here
// as a corpus gap and points at the wrong repository. [Coverage.String] prints
// the stamped set for exactly this reason: a gap with a populated stamped set
// reads differently from a gap with an empty one.
//
// patterns are Go package patterns relative to root, so a caller says
// "./corpus/..." regardless of where the test process happens to be running.
// The frontend resolves patterns against the working directory rather than
// against the pipeline's source root, so they are joined here — passing them
// through unchanged makes the result depend on which directory `go test` was
// invoked from.
func Annotate(ctx context.Context, root string, patterns ...string) (map[string][]string, error) {
	pipe, err := run(ctx, root, patterns...)
	if err != nil {
		return nil, err
	}
	return collect(pipe), nil
}

// run builds the annotation pipeline and drives it over the corpus, returning
// the pipeline so a caller can read the store it populated.
//
// Separate from [Annotate] because coverage is not the only thing worth
// measuring: [Resolution] reads the same store for a different property, and
// two constructions of this pipeline would be free to disagree about which
// annotators are registered — which is exactly the class of defect it exists to
// surface.
func run(ctx context.Context, root string, patterns ...string) (*pipeline.Pipeline, error) {
	// filepath.Join treats "..." as an ordinary path element, so a recursive
	// pattern survives the join unchanged.
	scoped := make([]string, len(patterns))
	for i, p := range patterns {
		scoped[i] = filepath.Join(root, p)
	}

	// Taken from the generator module rather than assembled here. The set the
	// gate measures and the set the CLI applies have to be the same one, and
	// two constructions of it would be free to drift. It also registers every
	// testkit directive schema, without which a fixture carrying one is
	// rejected as unknown rather than measured.
	builder := pipeline.New().
		WithBrand(brand.Name).
		WithDirectivePrefix(brand.DirectivePrefix).
		WithSourceRoot(root).
		WithFrontend(golang.New())
	for _, a := range generator.Annotators() {
		builder = builder.WithAnnotator(a)
	}

	pipe, err := builder.
		WithBackend(backendgolang.New()).
		WithSink(sink.NewMemory()).
		Build()
	if err != nil {
		return nil, fmt.Errorf("gate: build annotation pipeline: %w", err)
	}

	if err := pipe.Run(ctx, scoped...); err != nil {
		return nil, fmt.Errorf("gate: run annotation pipeline: %w", err)
	}

	return pipe, nil
}

// DetectorStamps returns each package's detector stamps — the per-fixture
// view [Annotate]'s corpus-wide census deliberately flattens, for the
// identity gate that holds a detector fixture to the shape its directory
// names. eidos#26 proved the need the hard way: two fixtures drifted from
// their named shapes with years of green builds, because coverage asked
// only whether the stamps existed somewhere.
func DetectorStamps(ctx context.Context, root string, patterns ...string) (map[string][]string, error) {
	pipe, err := run(ctx, root, patterns...)
	if err != nil {
		return nil, err
	}
	out := map[string][]string{}
	for _, iface := range pipe.Store().Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if s := shape.Get(m.Meta()); s != "" {
				out[iface.Package] = append(out[iface.Package], s)
			}
		}
	}
	return out, nil
}

// collect reads the stamped classifications off every callable in the store.
//
// Structural shape is a single value per callable, while contracts and mixins
// are lists — a callable carries one shape but may join several contracts and
// declare several mixins. The three are read through the shape package's own
// accessors rather than by reaching into the metadata bag, so a change to how
// eidos stores them does not silently return nothing here.
func collect(pipe *pipeline.Pipeline) map[string][]string {
	seen := map[string]map[string]struct{}{
		AxisDetector: {},
		AxisContract: {},
		AxisMixin:    {},
	}

	// Run has already returned successfully, so the store is populated. A nil
	// here would mean the pipeline reported success without producing one.
	store := pipe.Store()

	for _, m := range store.Nodes().Methods().Items() {
		bag := m.Meta()
		if s := shape.Get(bag); s != "" {
			seen[AxisDetector][s] = struct{}{}
		}
		for _, c := range shape.Contracts(bag) {
			seen[AxisContract][c] = struct{}{}
		}
		for _, x := range shape.Mixins(bag) {
			seen[AxisMixin][x] = struct{}{}
		}
	}

	// Free functions carry the same three axes. Skipping them would make any
	// classification only ever reachable outside an interface look uncovered.
	for _, f := range store.Nodes().Functions().Items() {
		bag := f.Meta()
		if s := shape.Get(bag); s != "" {
			seen[AxisDetector][s] = struct{}{}
		}
		for _, c := range shape.Contracts(bag) {
			seen[AxisContract][c] = struct{}{}
		}
		for _, x := range shape.Mixins(bag) {
			seen[AxisMixin][x] = struct{}{}
		}
	}

	return toSlices(seen)
}

// toSlices flattens the per-axis sets. [Compare] sorts what it receives, so
// map iteration order here is not load-bearing.
func toSlices(seen map[string]map[string]struct{}) map[string][]string {
	out := make(map[string][]string, len(seen))
	for axis, set := range seen {
		names := make([]string, 0, len(set))
		for n := range set {
			names = append(names, n)
		}
		out[axis] = names
	}
	return out
}
