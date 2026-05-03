// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"fmt"
	"path/filepath"
	"strings"

	"go.thesmos.sh/testkit/gen"
	"go.thesmos.sh/testkit/gen/directiveparse"
)

// Generator produces conformance spec suites for Go interfaces.
type Generator struct{}

// Name returns "suite".
func (*Generator) Name() string { return "suite" }

// Generate produces a spec file with AssertContract for the given interface.
func (*Generator) Generate(pkg *gen.Package, args []string, cfg gen.Config, opts gen.Options) (*gen.Result, error) {
	// 1. Validate: single arg, must be an interface.
	if errs := gen.ValidateTypes(pkg, args, gen.KindInterface); len(errs) > 0 {
		return nil, errs[0]
	}

	// 2. Analyze: build SpecData.
	data, err := Analyze(pkg, args, cfg, opts)
	if err != nil {
		return nil, err
	}

	// 3. Validate directives (strict-by-default).
	reg := directiveparse.DefaultRegistry()
	methods := make([]gen.MethodInfo, len(data.Methods))
	for i, m := range data.Methods {
		methods[i] = m.MethodInfo
	}
	if errs := reg.Validate(methods, nil); len(errs) > 0 {
		return nil, errs[0]
	}

	// 4. Enrich: run directive enrichers.
	enrichErr := Enrich(data, pkg)
	if enrichErr != nil {
		return nil, enrichErr
	}

	// 5. Validate composition.
	for _, m := range data.Methods {
		issues := directiveparse.ValidateComposition(m.Directives)
		for _, issue := range issues {
			if issue.Kind == directiveparse.Conflict || issue.Kind == directiveparse.MissingRequired {
				return nil, gen.Errorf(m.Pos, "%s", issue.Message)
			}
		}
	}

	// 6. Parse templates.
	tmpl, parseErr := gen.NewTemplateSet().ParseFS(templateFS, "templates/*.tmpl")
	if parseErr != nil {
		return nil, fmt.Errorf("parse templates: %w", parseErr)
	}

	// 7. Build source attribution.
	var sourceFile string
	if len(data.Methods) > 0 {
		minLine, maxLine := data.Methods[0].Pos.Line, data.Methods[0].Pos.Line
		filename := data.Methods[0].Pos.Filename
		for _, m := range data.Methods[1:] {
			if m.Pos.Line < minLine {
				minLine = m.Pos.Line
			}
			if m.Pos.Line > maxLine {
				maxLine = m.Pos.Line
			}
		}
		if filename != "" {
			sourceFile = fmt.Sprintf("%s:%d-%d", filepath.Base(filename), minLine, maxLine)
		}
	}

	header := gen.Header{
		Subcommand: "suite",
		Args:       "suite " + strings.Join(args, " "),
		SourceFile: sourceFile,
	}

	// 8. Render spec file.
	content, renderErr := gen.RenderTemplate(tmpl, "spec", data, header)
	if renderErr != nil {
		return nil, fmt.Errorf("render spec: %w", renderErr)
	}

	return &gen.Result{
		Files: []gen.OutputFile{
			{Path: opts.Output, Content: content},
		},
	}, nil
}
