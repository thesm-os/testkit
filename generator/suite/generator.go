// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"go/token"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	_ "go.thesmos.sh/testkit/generator/spec/all" // registers every shipped directive consumer
)

// pipeline configures the suite generator's [generator.Pipeline].
// One renderer emits the contract driver file into the consumer's
// test package — the same external _test conventions stub uses for
// its auto-test, but here the driver is the consumer-facing API.
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
			TemplateName: "suite",
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

// asTestView reshapes [Data] for the test renderer. The contract
// driver lives in the external _test package so PackageName /
// Imports / GenQualifier come from [generator.BuildTestFileInfo]
// — the source-package types resolve through the test file's
// alias (typically the source-pkg name itself).
//
// For generic interfaces, each MethodView's typeArgs is rewritten
// from "[T]" (param names) to "[int]" (concrete) so any reference
// the driver emits to per-method shape primitives renders the
// concrete forms templates expect.
func asTestView(d *Data, _ generator.Options) *Data {
	info := generator.BuildTestFileInfo(d.PackageName, d.Imports, generator.DefaultConfig(), d.ImplImportPath)
	out := *d
	out.PackageName = info.PackageName
	out.Imports = info.Imports
	out.GenQualifier = info.GenQualifier
	if d.Spec.IsGeneric {
		concrete := generator.TestTypeArgs(d.Spec.Interface.TypeParams)
		out.Methods = make([]MethodView, len(d.Methods))
		copy(out.Methods, d.Methods)
		for i := range out.Methods {
			out.Methods[i].typeArgs = concrete
		}
	}
	return &out
}

// Generator implements [generator.Generator] for the suite
// subcommand.
type Generator struct{}

// Name returns the subcommand name.
func (*Generator) Name() string { return pipeline.Name }

// Generate produces the contract driver for the requested interface.
func (*Generator) Generate(
	pkg *generator.Package, args []string, cfg generator.Config, opts generator.Options,
) (*generator.Result, error) {
	return pipeline.Run(pkg, args, cfg, opts)
}
