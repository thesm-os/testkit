// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package enum implements the enum test generator. It scans const
// blocks of named integer types and emits exhaustiveness, stringer,
// parse-round-trip, marshal-round-trip (text / JSON / binary), and
// wire-compat (G25) tests in a single _test.go file plus per-type
// JSON golden files for wire-compatibility detection.
package enum

import "go.thesmos.sh/testkit/generator"

// pipeline is the full enum generation strategy. Like sentinel, enum
// needs no directive consumers, no enrichment, no per-method
// composition checks — analyze the package, render templates,
// emit a test file plus per-type JSON goldens.
//
// Wire-compat goldens are emitted as additional [generator.OutputFile]
// entries from the post-render hook so a single Generate call writes
// every required artifact.
var pipeline = generator.Pipeline[*Data]{
	Name:      "enum",
	Kind:      generator.KindAny,
	Templates: templateFS,
	Analyze:   Analyze,
	Renderers: []generator.Renderer[*Data]{
		{TemplateName: "enum", Path: func(o generator.Options) string { return o.Output }},
	},
	PostRender: emitWireGolden,
}

// Generator implements [generator.Generator] for the enum subcommand.
type Generator struct{}

// Name returns the subcommand name.
func (*Generator) Name() string { return pipeline.Name }

// Generate runs the enum pipeline against pkg and returns the test
// file plus one wire-compat golden file per requested type. Returns
// an error when any requested type has no associated constants.
func (*Generator) Generate(
	pkg *generator.Package, args []string, cfg generator.Config, opts generator.Options,
) (*generator.Result, error) {
	return pipeline.Run(pkg, args, cfg, opts)
}
