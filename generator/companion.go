// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator

import (
	"go/types"
	"path/filepath"

	"go.thesmos.sh/testkit/generator/directive"
)

// LoadOutputPackage attempts to load the package the generator emits
// into, derived from [Options.Output] and [Options.WorkDir]. Returns
// nil for any failure mode — package not yet existing on first
// generation is the load-bearing case, not an error to propagate.
//
// Returns nil when Output or WorkDir is empty, when the output
// directory equals "." (same package as the source — already covered
// by the source-package lookup), or when the load itself fails (the
// directory typically doesn't exist on first generation).
func LoadOutputPackage(opts Options) *Package {
	if opts.Output == "" || opts.WorkDir == "" {
		return nil
	}
	outputDir := filepath.Dir(opts.Output)
	if outputDir == "." {
		return nil
	}
	loader := NewLoader()
	pkg, err := loader.Load(".", filepath.Join(opts.WorkDir, outputDir))
	if err != nil {
		return nil
	}
	return pkg
}

// LookupCompanionFunc searches the source package for a top-level
// function named `name` whose signature satisfies sigCheck. If
// absent, falls back to [LoadOutputPackage] and searches there —
// handling the chicken-and-egg where the companion function lives
// next to the generated code rather than next to the source type.
//
// The two return flags are independent: `found` reports whether the
// function was located at all; `fromOutput` reports whether the
// match came from the output package (true) or the source package
// (false). Callers use `fromOutput` to decide whether the rendered
// call expression needs a source-pkg qualifier prefix (false → yes,
// true → bare same-pkg reference).
func LookupCompanionFunc(
	srcPkg *Package, opts Options, name string, sigCheck func(*types.Signature) bool,
) (found, fromOutput bool) {
	if HasFunc(srcPkg, name, sigCheck) {
		return true, false
	}
	if outputPkg := LoadOutputPackage(opts); outputPkg != nil {
		if HasFunc(outputPkg, name, sigCheck) {
			return true, true
		}
	}
	return false, false
}

// FieldDirective returns the first //testkit:<name> directive
// attached to the given struct field, or (zero, false) if none.
// Pass a typed name constant from package [directive] (e.g.
// [directive.Default]) to keep the call site free of magic strings.
//
// Multiple occurrences of the same directive on one field are rare
// and currently undefined; callers needing all of them should walk
// [Package.FieldDirectives] directly.
func FieldDirective(pkg *Package, structName, fieldName, name string) (directive.Directive, bool) {
	for _, d := range pkg.FieldDirectives(structName, fieldName) {
		if d.Name == name {
			return d, true
		}
	}
	return directive.Directive{}, false
}
