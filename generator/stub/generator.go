// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub

import (
	"go/token"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	_ "go.thesmos.sh/testkit/generator/spec/all" // registers every shipped directive consumer
)

// pipeline configures the stub generator's [generator.Pipeline].
// Two renderers — impl + auto-test — share one Analyze pass; the
// test view's [Renderer.Transform] hook reshapes PackageName /
// GenQualifier / Imports for the external _test package.
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
			TemplateName: "stub",
			Path:         func(o generator.Options) string { return o.Output },
		},
		{
			TemplateName: "stub-test",
			Path:         func(o generator.Options) string { return generator.TestPathFrom(o.Output) },
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

// asTestView reshapes [Data] for the companion test renderer.
// PackageName / Imports / GenQualifier come from
// [generator.BuildTestFileInfo]; the impl-vs-test split makes the
// stub references resolve correctly across the package boundary.
func asTestView(d *Data, _ generator.Options) *Data {
	info := generator.BuildTestFileInfo(d.PackageName, d.Imports, generator.DefaultConfig(), d.ImplImportPath)
	out := *d
	out.PackageName = info.PackageName
	out.Imports = info.Imports
	out.GenQualifier = info.GenQualifier
	// For generic interfaces, the auto-test references the stub
	// with concrete instantiations — rewrite each MethodView's
	// typeArgs from "[T]" (param names) to "[int]" (concrete) so
	// QualStubType / QualCallType / QualReturnType render the
	// concrete forms templates emit.
	if d.Spec.IsGeneric {
		out.Methods = make([]MethodView, len(d.Methods))
		copy(out.Methods, d.Methods)
		for i := range out.Methods {
			out.Methods[i].typeArgs = d.TestTypeArgs
		}
	}
	return &out
}

// Generator implements [generator.Generator] for the stub
// subcommand.
type Generator struct{}

// Name returns the subcommand name.
func (*Generator) Name() string { return pipeline.Name }

// Generate produces the stub impl + companion test for the
// requested interface.
func (*Generator) Generate(
	pkg *generator.Package, args []string, cfg generator.Config, opts generator.Options,
) (*generator.Result, error) {
	return pipeline.Run(pkg, args, cfg, opts)
}
