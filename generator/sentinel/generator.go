// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sentinel

import "go.thesmos.sh/testkit/generator"

// pipeline is the full sentinel generation strategy. Sentinel needs
// no directive consumers, no enrichment, no per-method composition
// checks — just analyze the package and render one template. The
// pipeline absorbs every other step.
//
// [Data] satisfies [generator.SkippableData] via its HasContent
// method; the pipeline returns an empty [generator.Result] when the
// package has no Err* vars and no error types.
var pipeline = generator.Pipeline[*Data]{
	Name:      "sentinel",
	Kind:      generator.KindAny,
	Templates: templateFS,
	Analyze:   Analyze,
	Renderers: []generator.Renderer[*Data]{
		{TemplateName: "sentinel", Path: func(o generator.Options) string { return o.Output }},
	},
}

// Generator implements [generator.Generator] for the sentinel
// subcommand. It is a thin shell over [pipeline].
type Generator struct{}

// Name returns the subcommand name.
func (*Generator) Name() string { return pipeline.Name }

// Generate runs the sentinel pipeline against pkg and returns the
// emitted file (or an empty result when the package has no errors).
func (*Generator) Generate(
	pkg *generator.Package, args []string, cfg generator.Config, opts generator.Options,
) (*generator.Result, error) {
	return pipeline.Run(pkg, args, cfg, opts)
}
