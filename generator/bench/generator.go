// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package bench is the benchmark generator. It produces
// `Benchmark<Iface>Contract` drivers that wire every method of an
// interface into the [bench] package's per-shape primitives — always
// emitting HotPath + ConcurrentThroughput, plus opt-in
// [bench.AllocsWithin] / [bench.LatencyWithin] gates when
// `//testkit:allocs N` or `//testkit:latency D` are declared on the
// method.
//
// The driver also exposes a functional-option surface so consumers
// can plug in additional per-method primitives via
// `<Iface>BenchOn<Method>(...)`, supply a single `<Iface>BenchPrePopulate`
// seeder that runs against every freshly factory'd impl, or drop in
// arbitrary `<Iface>BenchCustom(name, fn)` sub-benchmarks the shape
// vocabulary doesn't cover.
package bench

import (
	"go/token"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	_ "go.thesmos.sh/testkit/generator/spec/all" // registers every shipped directive consumer
)

// pipeline configures the bench generator's [generator.Pipeline].
// One renderer emits the bench driver file into the consumer's test
// package — same external _test conventions stub/suite use.
var pipeline = generator.Pipeline[*Data]{
	Name:                 Name,
	Kind:                 generator.KindInterface,
	Templates:            templateFS,
	Analyze:              Analyze,
	Enrich:               Enrich,
	PostEnrich:           Project,
	Methods:              methodsFor,
	Positions:            positionsFor,
	CompositionValidator: directive.ValidateComposition,
	Renderers: []generator.Renderer[*Data]{
		{
			TemplateName: "bench",
			Path:         func(o generator.Options) string { return o.Output },
			Transform:    asTestView,
		},
	},
}

// methodsFor surfaces the spec-level [generator.MethodInfo] slice
// to the pipeline's DirectiveValidator + CompositionValidator. Both
// validators run BEFORE Enrich populates Attachments — they only
// inspect directive *names*, not consumer payloads.
func methodsFor(d *Data) []generator.MethodInfo {
	out := make([]generator.MethodInfo, len(d.Spec.Methods))
	for i, m := range d.Spec.Methods {
		out[i] = m.MethodInfo
	}
	return out
}

// positionsFor extracts source positions for the SourceAttribution
// header.
func positionsFor(d *Data) []token.Position {
	out := make([]token.Position, len(d.Spec.Methods))
	for i, m := range d.Spec.Methods {
		out[i] = m.Pos
	}
	return out
}

// asTestView reshapes [Data] for the renderer. Unlike suite (which
// emits an external `_test` package), bench output lives in the same
// package as the impl's generated companions — `package storetest`,
// not `package storetest_test`. The bench driver is a regular
// function callable from any benchmark file, so the test-package
// reshape would only force callers to import a `_test` pkg they
// can't reach.
//
// The transform handles two concerns:
//
//  1. PackageName comes straight from the output-path-derived value
//     in [generator.BuildOutputCtx]; we don't reshape it.
//  2. For generic interfaces, the driver is monomorphized: each
//     MethodView's substitute closure rewrites type-parameter names
//     to the concrete instantiation. Without the substitute, the
//     driver body would mix abstract V against the concrete
//     `Holder[string]` factory and fail to compile.
func asTestView(d *Data, _ generator.Options) *Data {
	out := *d
	if d.Spec.IsGeneric {
		substitute := func(s string) string {
			return generator.SubstituteTypeParams(s, d.Spec.Interface.TypeParams)
		}
		out.Methods = make([]MethodView, len(d.Methods))
		copy(out.Methods, d.Methods)
		for i := range out.Methods {
			out.Methods[i].substitute = substitute
		}
	}
	return &out
}

// Generator implements [generator.Generator] for the bench
// subcommand.
type Generator struct{}

// Name returns the subcommand name.
func (*Generator) Name() string { return pipeline.Name }

// Generate produces the bench driver for the requested interface.
func (*Generator) Generate(
	pkg *generator.Package, args []string, cfg generator.Config, opts generator.Options,
) (*generator.Result, error) {
	return pipeline.Run(pkg, args, cfg, opts)
}
