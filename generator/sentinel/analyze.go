// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sentinel

import (
	"go/token"
	"go/types"
	"path"
	"strings"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
)

// errorTypeName is the rendered string for the builtin error
// interface — used to detect error-typed fields without importing
// go/types into the data layer.
const errorTypeName = "error"

// errPrefix is the name prefix that distinguishes a sentinel error
// variable from any other exported package-level var.
const errPrefix = "Err"

// scanErrorVars returns every exported package-level Err* var in pkg,
// optionally restricted to sourceFile. Thin wrapper over
// [generator.ScanVars].
func scanErrorVars(pkg *generator.Package, sourceFile string) []*generator.VarInfo {
	return generator.ScanVars(pkg, sourceFile, func(v *types.Var) bool {
		return strings.HasPrefix(v.Name(), errPrefix)
	})
}

// scanErrorTypes returns every exported struct in pkg whose pointer
// receiver implements the error interface. Thin wrapper over
// [generator.ScanStructsImplementing].
func scanErrorTypes(pkg *generator.Package) []*generator.StructInfo {
	return generator.ScanStructsImplementing(pkg, generator.ErrorInterface())
}

// Analyze produces a [Data] from a loaded package. Returns a Data
// with HasContent()==false when the package has no sentinels and no
// error types; the caller (the [Generator]) short-circuits in that
// case.
//
// Cross-package overlap data is populated when the package declares
// `//testkit:sentinel-no-overlap-with <import>` in its package doc
// comment. Each named package is loaded via a fresh [generator.Loader]
// scoped to opts.WorkDir so the same caching applies.
func Analyze(pkg *generator.Package, _ []string, cfg generator.Config, opts generator.Options) (*Data, error) {
	ctx, err := generator.BuildOutputCtx(pkg, cfg, opts)
	if err != nil {
		return nil, err
	}

	vars := scanErrorVars(pkg, opts.SourceFile)
	sentinels := make([]ErrorVar, len(vars))
	for i, v := range vars {
		sentinels[i] = ErrorVar{Name: v.Name}
	}

	crossPkgs, err := buildCrossPackages(pkg, opts)
	if err != nil {
		return nil, err
	}

	return &Data{
		PackageName:   ctx.PackageName,
		ImportPath:    ctx.ImportPath,
		Qualifier:     ctx.Qualifier,
		TestName:      generator.CamelCase(pkg.Name()) + "SentinelErrors",
		Prefix:        pkg.Name() + ": ",
		Sentinels:     sentinels,
		ErrorTypes:    buildErrorTypes(pkg, ctx.Tracker, ctx.Qualifier),
		CrossPackages: crossPkgs,
		Directives: generator.RenderPackageDirectives(pkg,
			directive.SentinelNoOverlapWith),
	}, nil
}

// buildErrorTypes scans the package for custom error types and renders
// per-field sample data for each. The qualifier propagates to each
// ErrorType so partial templates can render type references without
// parent context. OtherTypes and FormatCheckOrder are filled in a
// second pass once the full set of names is known.
func buildErrorTypes(pkg *generator.Package, tracker *generator.ImportTracker, qualifier string) []ErrorType {
	structs := scanErrorTypes(pkg)
	out := make([]ErrorType, 0, len(structs))
	for _, s := range structs {
		et := ErrorType{
			Name:      s.Name,
			Qualifier: qualifier,
			HasIs:     generator.HasMethod(pkg, s.Name, "Is", generator.IsErrorBoolSig),
			HasUnwrap: generator.HasMethod(pkg, s.Name, "Unwrap", generator.UnwrapSig),
		}
		for _, f := range s.Fields {
			if !f.Exported {
				continue
			}
			fd := FieldData{Name: f.Name, TypeStr: f.Type.String()}
			lowerName := strings.ToLower(f.Name)
			switch f.Type.String() {
			case errorTypeName:
				// Use a real error so Unwrap/errors.Is chains have
				// something distinguishable to find.
				fd.SampleValue = `errors.New("test-` + lowerName + `")`
				fd.FormatCheckValue = "test-" + lowerName
				fd.IsError = true
				if et.HasUnwrap && et.UnwrapField == "" {
					et.UnwrapField = f.Name
				}
			case "string":
				fd.SampleValue = generator.SampleValueOf(f.Type, f.Name, tracker)
				fd.FormatCheckValue = "test-" + lowerName
			default:
				fd.SampleValue = generator.SampleValueOf(f.Type, f.Name, tracker)
			}
			et.Fields = append(et.Fields, fd)
		}
		out = append(out, et)
	}

	// Pre-compute OtherTypes (cross-error-type non-overlap subtests)
	// and FormatCheckOrder (format-strictness assertion) so templates
	// stay free of slice-building primitives.
	for i := range out {
		for j := range out {
			if i == j {
				continue
			}
			out[i].OtherTypes = append(out[i].OtherTypes, out[j].Name)
		}
		for _, f := range out[i].Fields {
			if f.FormatCheckValue != "" {
				out[i].FormatCheckOrder = append(out[i].FormatCheckOrder, f.FormatCheckValue)
			}
		}
	}
	return out
}

// buildCrossPackages reads the package-level
// //testkit:sentinel-no-overlap-with directive(s) and loads the
// referenced packages, scanning each for its own Err* sentinels.
//
// Loading uses a fresh [generator.Loader] scoped to
// [generator.Options.WorkDir] so module resolution matches the local
// build. Errors during peer loading bubble up as Analyze errors —
// silently dropping a misconfigured directive would mask the very
// bugs G24 is trying to catch.
func buildCrossPackages(pkg *generator.Package, opts generator.Options) ([]CrossPackage, error) {
	var imports []string
	for _, d := range pkg.PackageDirectives() {
		if d.Name != directive.SentinelNoOverlapWith {
			continue
		}
		imports = append(imports, d.Args...)
	}
	if len(imports) == 0 {
		return nil, nil
	}

	loader := generator.NewLoader()
	out := make([]CrossPackage, 0, len(imports))
	for _, imp := range imports {
		peer, err := loader.Load(imp, opts.WorkDir)
		if err != nil {
			return nil, generator.WrapErr(token.Position{}, err,
				"sentinel-no-overlap-with: load %q", imp)
		}
		peerVars := scanErrorVars(peer, "")
		if len(peerVars) == 0 {
			// Don't emit cross-pair subtests for a peer without any
			// sentinels — the assertion would be vacuous.
			continue
		}
		peerSentinels := make([]ErrorVar, len(peerVars))
		for i, v := range peerVars {
			peerSentinels[i] = ErrorVar{Name: v.Name}
		}
		out = append(out, CrossPackage{
			ImportPath: imp,
			Alias:      path.Base(imp),
			Sentinels:  peerSentinels,
		})
	}
	return out, nil
}
