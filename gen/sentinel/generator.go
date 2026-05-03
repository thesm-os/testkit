// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sentinel

import (
	"fmt"

	"go.thesmos.sh/testkit/gen"
)

const generatorName = "sentinel"

// Generator produces sentinel error tests for Go packages.
type Generator struct{}

// Name returns generatorName.
func (*Generator) Name() string { return generatorName }

// Generate scans the package for exported Err* variables and produces
// a test file that verifies error message consistency, uniqueness,
// non-overlap, and unwrap chain preservation.
func (*Generator) Generate(
	pkg *gen.Package,
	_ []string,
	cfg gen.Config,
	opts gen.Options,
) (*gen.Result, error) {
	data, err := Analyze(pkg, cfg, opts)
	if err != nil {
		return nil, err
	}

	if !data.HasContent() {
		return &gen.Result{}, nil
	}

	tmpl, parseErr := gen.NewTemplateSet().ParseFS(templateFS, "templates/*.tmpl")
	if parseErr != nil {
		return nil, fmt.Errorf("parse templates: %w", parseErr)
	}

	header := gen.Header{
		Subcommand: generatorName,
		Args:       generatorName,
	}

	content, renderErr := gen.RenderTemplate(tmpl, generatorName, data, header)
	if renderErr != nil {
		return nil, fmt.Errorf("render sentinel: %w", renderErr)
	}

	return &gen.Result{
		Files: []gen.OutputFile{
			{Path: opts.Output, Content: content},
		},
	}, nil
}
