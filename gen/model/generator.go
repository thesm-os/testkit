// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package model implements the model-based testing generator for testkit.
// It produces AssertXxxModel functions with shape-derived state machine
// actions, auto-laws, and reference synthesis from shape detection.
//
// Reuses the suite package's Analyze and Enrich pipeline — same shape
// detection, same directive parsing — and renders model-specific templates.
package model

import (
	"fmt"
	"path/filepath"
	"strings"

	"go.thesmos.sh/testkit/gen"
	"go.thesmos.sh/testkit/gen/directiveparse"
	"go.thesmos.sh/testkit/gen/suite"
)

// Generator produces model-based test harnesses for Go interfaces.
type Generator struct{}

// Name returns "model".
func (*Generator) Name() string { return "model" }

// Generate produces a model file with AssertModel for the given interface.
func (*Generator) Generate(pkg *gen.Package, args []string, cfg gen.Config, opts gen.Options) (*gen.Result, error) {
	if errs := gen.ValidateTypes(pkg, args, gen.KindInterface); len(errs) > 0 {
		return nil, errs[0]
	}

	data, err := suite.Analyze(pkg, args, cfg, opts)
	if err != nil {
		return nil, err
	}

	reg := directiveparse.DefaultRegistry()
	methods := make([]gen.MethodInfo, len(data.Methods))
	for i, m := range data.Methods {
		methods[i] = m.MethodInfo
	}
	if errs := reg.Validate(methods, nil); len(errs) > 0 {
		return nil, errs[0]
	}

	enrichErr := suite.Enrich(data, pkg)
	if enrichErr != nil {
		return nil, enrichErr
	}

	for _, m := range data.Methods {
		issues := directiveparse.ValidateComposition(m.Directives)
		for _, issue := range issues {
			if issue.Kind == directiveparse.Conflict || issue.Kind == directiveparse.MissingRequired {
				return nil, gen.Errorf(m.Pos, "%s", issue.Message)
			}
		}
	}

	// Build model-specific data.
	md := buildData(data)

	tmplSet := gen.NewTemplateSet()
	tmplSet.Funcs(md.templateFuncs())
	tmpl, parseErr := tmplSet.ParseFS(templateFS, "templates/*.tmpl")
	if parseErr != nil {
		return nil, fmt.Errorf("parse templates: %w", parseErr)
	}

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
		Subcommand: "model",
		Args:       "model " + strings.Join(args, " "),
		SourceFile: sourceFile,
	}

	content, renderErr := gen.RenderTemplate(tmpl, "model", md, header)
	if renderErr != nil {
		return nil, fmt.Errorf("render model: %w", renderErr)
	}

	return &gen.Result{
		Files: []gen.OutputFile{
			{Path: opts.Output, Content: content},
		},
	}, nil
}
