// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import "strings"

const (
	sentinelGenerator     = "sentinel"
	sentinelTmplFile      = "sentinel.go.tmpl"
	sentinelDefaultOutput = "errors.gen_test.go"

	errorsFilePrefix = "errors_"
	errorsFileName   = "errors"
	goSuffix         = ".go"
)

// GenerateSentinel produces tests for all exported Err* variables in a
// package (or in a single source file when Options.SourceFile is set via
// $GOFILE). Tests verify prefix consistency, uniqueness, non-overlap, and
// unwrap chain preservation. Returns a [Result] with a single test file.
func GenerateSentinel(pkg *Package, cfg Config, opts Options) (*Result, error) {
	vars := pkg.ErrorVars(opts.SourceFile)
	if len(vars) == 0 {
		return nil, Errorf(emptyPos, "no exported Err* variables found in package %s", pkg.Pkg.Name())
	}

	outputPath := opts.Output
	if outputPath == "" {
		outputPath = sentinelDefaultOutput
	}

	pkgName := DerivePackageName(outputPath, pkg.Pkg.Name(), cfg)

	isExternal := pkgName != pkg.Pkg.Name()
	qualifier := ""
	importPath := ""
	if isExternal {
		qualifier = pkg.Pkg.Name()
		importPath = pkg.Pkg.Path()
	}

	testName := deriveSentinelTestName(pkg.Pkg.Name(), opts.SourceFile)

	var sentinels []sentinelData
	for _, v := range vars {
		sentinels = append(sentinels, sentinelData{
			Name: v.Name,
			Doc:  v.Doc,
		})
	}

	data := sentinelTemplateData{
		PackageName:  pkgName,
		TestName:     testName,
		Prefix:       pkg.Pkg.Name() + ": ",
		PkgQualifier: qualifier,
		ImportPath:   importPath,
		Sentinels:    sentinels,
	}

	header := Header{
		Subcommand: sentinelGenerator,
		Args:       sentinelGenerator,
	}

	content, err := Render(templateFile(sentinelTmplFile), data, header)
	if err != nil {
		return nil, WrapErr(emptyPos, err, "render sentinel tests")
	}

	return &Result{
		Files: []OutputFile{
			{Path: outputPath, Content: content},
		},
	}, nil
}

// deriveSentinelTestName produces the test function name.
// Package "store" → "StoreSentinel".
// Package "store" + file "errors_auth.go" → "StoreAuthSentinel".
func deriveSentinelTestName(pkgName, sourceFile string) string {
	base := Title(pkgName)
	if sourceFile == "" {
		return base + "Sentinel"
	}
	filePart := strings.TrimSuffix(sourceFile, goSuffix)
	filePart = strings.TrimPrefix(filePart, errorsFilePrefix)
	filePart = strings.TrimPrefix(filePart, errorsFileName)
	if filePart == "" {
		return base + "Sentinel"
	}
	return base + Title(filePart) + "Sentinel"
}

// --- data types ---

type sentinelData struct {
	Name string // "ErrNotFound"
	Doc  string
}

type sentinelTemplateData struct {
	PackageName  string
	TestName     string // "StoreSentinel", "StoreAuthSentinel"
	Prefix       string // "store: "
	PkgQualifier string // "store" — empty for internal test package
	ImportPath   string // import path for source package — empty for internal
	Sentinels    []sentinelData
}
