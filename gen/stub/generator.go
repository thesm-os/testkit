// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub

import (
	"fmt"
	"strings"

	"go.thesmos.sh/testkit/gen"
	"go.thesmos.sh/testkit/gen/directive"
)

// Generator produces stub test doubles for Go interfaces.
type Generator struct{}

// Name returns "stub".
func (*Generator) Name() string { return "stub" }

// Generate produces stub implementation and test files for the given
// interface types. Returns a Result with two OutputFiles: the stub
// implementation and the companion test file.
func (*Generator) Generate(pkg *gen.Package, args []string, cfg gen.Config, opts gen.Options) (*gen.Result, error) {
	// 1. Validate: all args must be interfaces.
	if errs := gen.ValidateTypes(pkg, args, gen.KindInterface); len(errs) > 0 {
		return nil, errs[0]
	}

	// 2. Analyze: build Data.
	data, err := Analyze(pkg, args, cfg, opts)
	if err != nil {
		return nil, err
	}

	// 3. Enrich: run directive enrichers.
	enrichErr := Enrich(data, pkg)
	if enrichErr != nil {
		return nil, enrichErr
	}

	// 4. Validate composition.
	for i := range data.Interfaces {
		for _, m := range data.Interfaces[i].Methods {
			issues := directive.ValidateComposition(m.Directives)
			for _, issue := range issues {
				if issue.Kind == directive.Conflict || issue.Kind == directive.MissingRequired {
					return nil, gen.Errorf(m.Pos, "%s", issue.Message)
				}
			}
		}
	}

	// 5. Parse templates.
	tmpl, parseErr := gen.NewTemplateSet().ParseFS(templateFS, "templates/*.tmpl")
	if parseErr != nil {
		return nil, fmt.Errorf("parse templates: %w", parseErr)
	}

	header := gen.Header{
		Subcommand: "stub",
		Args:       "stub " + strings.Join(args, " "),
	}

	// 6. Render stub file.
	stubContent, renderErr := gen.RenderTemplate(tmpl, "stub", data, header)
	if renderErr != nil {
		return nil, fmt.Errorf("render stub: %w", renderErr)
	}

	// 7. Build test data and render test file.
	stubImportPath, pathErr := gen.OutputImportPath(opts.Output, pkg)
	if pathErr != nil {
		return nil, fmt.Errorf("compute stub import path: %w", pathErr)
	}
	testData := buildTestData(data, cfg, stubImportPath)
	testContent, testRenderErr := gen.RenderTemplate(tmpl, "test", testData, header)
	if testRenderErr != nil {
		return nil, fmt.Errorf("render test: %w", testRenderErr)
	}

	implPath := opts.Output
	testPath := gen.TestPathFrom(implPath)

	return &gen.Result{
		Files: []gen.OutputFile{
			{Path: implPath, Content: stubContent},
			{Path: testPath, Content: testContent},
		},
	}, nil
}

// buildTestData creates a Data copy for the test file template.
// For external test style, the package name gets _test appended and
// references to stub types are qualified with the stub package name.
// The import list is rebuilt to include the stub package itself since
// goimports cannot auto-discover testdata packages.
func buildTestData(data *Data, cfg gen.Config, stubImportPath string) *Data {
	testPkgName := data.PackageName
	genQualifier := ""

	// Start with the stub file's imports.
	imports := make([]gen.Import, len(data.Imports))
	copy(imports, data.Imports)

	if cfg.TestPackageStyle == gen.TestPackageStyleExternal {
		testPkgName = data.PackageName + gen.TestPkgSuffix
		genQualifier = data.PackageName + "."

		// Add the stub package itself to the import list.
		imports = append(imports, gen.Import{Path: stubImportPath})
	}

	return &Data{
		PackageName:  testPkgName,
		Imports:      imports,
		Interfaces:   data.Interfaces,
		GenQualifier: genQualifier,
	}
}
