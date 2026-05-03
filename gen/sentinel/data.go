// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package sentinel implements the sentinel generator for testkit. It scans
// a package for exported Err* variables and custom error types, generating
// tests that verify error message consistency, uniqueness, non-overlap,
// unwrap chain preservation, and errors.As type matching.
package sentinel

import (
	"strings"

	"go.thesmos.sh/testkit/gen"
)

// Data is the top-level template data for a sentinel generation run.
type Data struct {
	PackageName  string
	ImportPath   string // source package import path (empty if same package)
	PkgQualifier string // "store" or "" for same package
	TestName     string // "SentinelErrors"
	Prefix       string // "store: " — expected error message prefix
	Sentinels    []ErrorVar
	ErrorTypes   []ErrorType
}

// ErrorVar holds one exported Err* variable.
type ErrorVar struct {
	Name string // "ErrNotFound"
	Doc  string // doc comment
}

const errorTypeName = "error"

// ErrorType holds one exported type implementing the error interface.
type ErrorType struct {
	Name        string      // "NotFoundError"
	Fields      []FieldData // exported fields
	HasIs       bool        // has Is(error) bool method
	HasUnwrap   bool        // has Unwrap() error method
	UnwrapField string      // "Cause" — the error field Unwrap returns (if HasUnwrap)
}

// FieldData holds one exported field of an error type.
type FieldData struct {
	Name             string // "ID"
	TypeStr          string // "string"
	SampleValue      string // `"test-id"` or `errors.New("test-cause")`
	FormatCheckValue string // "test-id" — plain string to check in Error() output
	IsError          bool   // true if field type is error (use ErrorIs, not Equal)
}

// HasContent reports whether there are any sentinels or error types to test.
func (d *Data) HasContent() bool {
	return len(d.Sentinels) > 0 || len(d.ErrorTypes) > 0
}

// Analyze builds Data by scanning the package for exported Err* variables
// and types implementing the error interface.
func Analyze(pkg *gen.Package, cfg gen.Config, opts gen.Options) (*Data, error) {
	vars := pkg.ErrorVars(opts.SourceFile)

	sentinels := make([]ErrorVar, len(vars))
	for i, v := range vars {
		sentinels[i] = ErrorVar{
			Name: v.Name,
			Doc:  v.Doc,
		}
	}

	pkgName := gen.DerivePackageName(opts.Output, pkg.Pkg.Name(), cfg)
	testName := gen.CamelCase(pkg.Pkg.Name()) + "SentinelErrors"

	// Compute import path and qualifier. The source package must be
	// imported when the output is in a different package — either a
	// subdirectory or a same-directory _test package.
	var importPath, qualifier string
	outputImportPath, err := gen.OutputImportPath(opts.Output, pkg)
	if err != nil {
		return nil, err
	}
	needsImport := outputImportPath != pkg.Pkg.Path() ||
		pkgName != pkg.Pkg.Name()
	if needsImport {
		importPath = pkg.Pkg.Path()
		qualifier = pkg.Pkg.Name()
	}

	// Derive prefix from package name.
	prefix := pkg.Pkg.Name() + ": "

	// Scan for custom error types.
	tracker := gen.NewImportTracker(outputImportPath)
	if importPath != "" {
		tracker.AddPath(importPath)
	}
	errorStructs := pkg.ErrorTypes()
	errorTypes := make([]ErrorType, 0, len(errorStructs))
	for _, s := range errorStructs {
		et := ErrorType{
			Name:      s.Name,
			HasIs:     pkg.ErrorTypeHasIs(s.Name),
			HasUnwrap: pkg.ErrorTypeHasUnwrap(s.Name),
		}
		for _, f := range s.Fields {
			if !f.Exported {
				continue
			}
			sample := gen.SampleValueOf(f.Type, f.Name, tracker)
			formatCheck := ""
			if f.Type.String() == errorTypeName {
				// For error-typed fields, use a real error instead of nil.
				lowerName := strings.ToLower(f.Name)
				sample = `errors.New("test-` + lowerName + `")`
				formatCheck = "test-" + lowerName
			} else if f.Type.String() == "string" {
				// For string fields, the format check is the unquoted value.
				formatCheck = "test-" + strings.ToLower(f.Name)
			}
			et.Fields = append(et.Fields, FieldData{
				Name:             f.Name,
				TypeStr:          f.Type.String(),
				SampleValue:      sample,
				FormatCheckValue: formatCheck,
				IsError:          f.Type.String() == errorTypeName,
			})
			// Find the error-typed field for Unwrap tests.
			if et.HasUnwrap && f.Type.String() == errorTypeName {
				et.UnwrapField = f.Name
			}
		}
		errorTypes = append(errorTypes, et)
	}

	return &Data{
		PackageName:  pkgName,
		ImportPath:   importPath,
		PkgQualifier: qualifier,
		TestName:     testName,
		Prefix:       prefix,
		Sentinels:    sentinels,
		ErrorTypes:   errorTypes,
	}, nil
}
