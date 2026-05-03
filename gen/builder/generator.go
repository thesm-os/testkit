// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"fmt"
	"strings"

	"go.thesmos.sh/testkit/gen"
)

// Generator produces fluent builders for Go structs.
type Generator struct{}

// Name returns "builder".
func (*Generator) Name() string { return "builder" }

// Generate produces a builder file and companion test file for the
// given struct types.
func (*Generator) Generate(
	pkg *gen.Package,
	args []string,
	cfg gen.Config,
	opts gen.Options,
) (*gen.Result, error) {
	// 1. Validate: all args must be structs.
	if errs := gen.ValidateTypes(pkg, args, gen.KindStruct); len(errs) > 0 {
		return nil, errs[0]
	}

	// 2. Analyze.
	data, err := Analyze(pkg, args, cfg, opts)
	if err != nil {
		return nil, err
	}

	// 3. Parse templates.
	tmpl, parseErr := gen.NewTemplateSet().ParseFS(templateFS, "templates/*.tmpl")
	if parseErr != nil {
		return nil, fmt.Errorf("parse templates: %w", parseErr)
	}

	header := gen.Header{
		Subcommand: "builder",
		Args:       "builder " + strings.Join(args, " "),
	}

	// 4. Render builder file.
	builderContent, renderErr := gen.RenderTemplate(tmpl, "builder", data, header)
	if renderErr != nil {
		return nil, fmt.Errorf("render builder: %w", renderErr)
	}

	// 5. Render test file.
	builderImportPath, pathErr := gen.OutputImportPath(opts.Output, pkg)
	if pathErr != nil {
		return nil, fmt.Errorf("compute builder import path: %w", pathErr)
	}
	testData := buildTestData(data, cfg, builderImportPath)
	testContent, testErr := gen.RenderTemplate(tmpl, "builder-test", testData, header)
	if testErr != nil {
		return nil, fmt.Errorf("render test: %w", testErr)
	}

	implPath := opts.Output
	testPath := gen.TestPathFrom(implPath)

	return &gen.Result{
		Files: []gen.OutputFile{
			{Path: implPath, Content: builderContent},
			{Path: testPath, Content: testContent},
		},
	}, nil
}

func buildTestData(data *Data, cfg gen.Config, genImportPath string) *Data {
	info := gen.BuildTestFileInfo(data.PackageName, data.Imports, cfg, genImportPath)
	return &Data{
		PackageName:  info.PackageName,
		Imports:      info.Imports,
		Structs:      data.Structs,
		GenQualifier: info.GenQualifier,
	}
}
