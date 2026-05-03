// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enum

import (
	"fmt"
	"strings"

	"go.thesmos.sh/testkit/gen"
)

const generatorName = "enum"

// Generator produces exhaustiveness tests for Go enum types.
type Generator struct{}

// Name returns "enum".
func (*Generator) Name() string { return generatorName }

// Generate scans const blocks for the given type names and produces
// a test file with exhaustiveness, distinctness, and stringer tests.
func (*Generator) Generate(
	pkg *gen.Package,
	args []string,
	cfg gen.Config,
	opts gen.Options,
) (*gen.Result, error) {
	data, err := Analyze(pkg, args, cfg, opts)
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
		Args:       generatorName + " " + strings.Join(args, " "),
	}

	content, renderErr := gen.RenderTemplate(tmpl, generatorName, data, header)
	if renderErr != nil {
		return nil, fmt.Errorf("render enum: %w", renderErr)
	}

	return &gen.Result{
		Files: []gen.OutputFile{
			{Path: opts.Output, Content: content},
		},
	}, nil
}
