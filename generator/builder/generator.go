// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package builder generates fluent test-fixture builders for Go
// struct types. Each Generate call emits two files: the impl (the
// builders themselves) and the companion test file (verifying the
// contract every builder honors).
package builder

import (
	"go.thesmos.sh/testkit/generator"
)

// pipeline is the builder generation strategy. Two renderers
// against the same Data — the impl from the "builder" template,
// the companion test from "builder-test" with a reshaped Data
// (different package + GenQualifier when the test lives in a
// sibling package, computed via [generator.BuildTestFileInfo]).
//
// Builder consumes no //testkit: directives at the package or
// method level — only the field-level //testkit:default which is
// resolved during analyze. That keeps DirectiveValidator /
// CompositionValidator slots empty.
var pipeline = generator.Pipeline[*Data]{
	Name:      "builder",
	Kind:      generator.KindStruct,
	Templates: templateFS,
	Analyze:   Analyze,
	Renderers: []generator.Renderer[*Data]{
		{
			TemplateName: "builder",
			Path:         func(o generator.Options) string { return o.Output },
		},
		{
			TemplateName: "builder-test",
			Path:         func(o generator.Options) string { return generator.TestPathFrom(o.Output) },
			Transform:    asTestView,
		},
	},
}

// asTestView reshapes Data for the test-file render. The test file
// usually lives in a sibling package (the generator's output is
// `<pkg>test/builders.gen.go` plus `<pkg>test/builders.gen_test.go`),
// so PackageName, Imports, and GenQualifier are recomputed via
// [generator.BuildTestFileInfo]. The impl import path travels
// through Data (resolved at analyze time via BuildOutputCtx).
//
// Output-pkg defaults factories also need rewriting: the impl
// renders them bare (same package), but from the external _test
// view the call must be GenQualifier-prefixed to reach across
// the package boundary. Source-pkg factories already carry their
// source qualifier so they render unchanged in either view.
func asTestView(d *Data, _ generator.Options) *Data {
	info := generator.BuildTestFileInfo(d.PackageName, d.Imports, generator.DefaultConfig(), d.ImplImportPath)
	structs := make([]StructData, len(d.Structs))
	for i, s := range d.Structs {
		if s.DefaultsFromOutput {
			s.DefaultsFunc = info.GenQualifier + s.DefaultsFunc
		}
		structs[i] = s
	}
	return &Data{
		PackageName:    info.PackageName,
		Imports:        info.Imports,
		Structs:        structs,
		GenQualifier:   info.GenQualifier,
		ImplImportPath: d.ImplImportPath,
		Directives:     d.Directives,
	}
}

// Generator implements [generator.Generator] for the builder
// subcommand.
type Generator struct{}

// Name returns the subcommand name.
func (*Generator) Name() string { return pipeline.Name }

// Generate produces the builder file and companion test file for
// the requested struct types.
func (*Generator) Generate(
	pkg *generator.Package, args []string, cfg generator.Config, opts generator.Options,
) (*generator.Result, error) {
	return pipeline.Run(pkg, args, cfg, opts)
}
