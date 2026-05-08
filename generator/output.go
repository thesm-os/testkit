// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator

import (
	"errors"
	"path/filepath"
	"strings"
)

// ValidateTypes checks that all named types in args exist in pkg and
// satisfy the requested [TypeKind]. Returns a slice of positioned
// errors — empty when all types validate.
//
// Generators call this as the first step of their pipeline so type-
// not-found and wrong-kind errors fail before any analysis runs.
func ValidateTypes(pkg *Package, args []string, kind TypeKind) []*Error {
	var errs []*Error
	for _, arg := range args {
		obj := pkg.Pkg.Scope().Lookup(arg)
		if obj == nil {
			errs = append(errs, Errorf(noPos,
				"type %q not found in package %s", arg, pkg.Pkg.Name()))
			continue
		}
		switch kind {
		case KindAny:
			continue
		case KindInterface:
			if _, err := pkg.Interface(arg); err != nil {
				var e *Error
				if errors.As(err, &e) {
					errs = append(errs, e)
				} else {
					errs = append(errs, Errorf(noPos, "%v", err))
				}
			}
		case KindStruct:
			if _, err := pkg.Struct(arg); err != nil {
				var e *Error
				if errors.As(err, &e) {
					errs = append(errs, e)
				} else {
					errs = append(errs, Errorf(noPos, "%v", err))
				}
			}
		case KindNamedType:
			// Any named type is acceptable; just verify the lookup succeeded.
		}
	}
	return errs
}

// OutputImportPath computes the Go import path for a generated file
// given its output path (relative to the source package's directory)
// and the source [*Package]. Used when one generator's output needs
// to import another's (e.g., a test file importing the stub).
//
// When [Options.OutputImportBase] is set (the CLI's -p flag has
// loaded a remote package), the path is computed relative to the
// CWD's import path rather than the source package's module — the
// output lands in the CWD, not the remote source.
func OutputImportPath(outputPath string, pkg *Package, opts Options) (string, error) {
	if opts.OutputImportBase != "" {
		base := opts.OutputImportBase
		dir := filepath.Dir(outputPath)
		dir = filepath.ToSlash(dir)
		if dir == "." || dir == "" {
			return base, nil
		}
		return base + "/" + dir, nil
	}
	if pkg == nil || pkg.Module == nil {
		return "", Errorf(noPos, "package has no module info; cannot compute import path")
	}
	abs, err := filepath.Abs(filepath.Join(opts.WorkDir, outputPath))
	if err != nil {
		return "", WrapErr(noPos, err, "resolve output path")
	}
	dir := filepath.Dir(abs)

	rel, err := filepath.Rel(pkg.Module.Dir, dir)
	if err != nil {
		return "", WrapErr(noPos, err, "compute relative path")
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return pkg.Module.Path, nil
	}
	return pkg.Module.Path + "/" + rel, nil
}

// TestPathFrom derives the *_test.go path from an implementation path:
//
//	"storetest/store_stub.gen.go"   → "storetest/store_stub.gen_test.go"
//	"storetest/store_stub.gen_test.go" → unchanged
func TestPathFrom(implPath string) string {
	if strings.HasSuffix(implPath, "_test.go") {
		return implPath
	}
	for _, suf := range []string{".gen.go", ".go"} {
		if base, ok := strings.CutSuffix(implPath, suf); ok {
			return base + strings.TrimSuffix(suf, ".go") + "_test.go"
		}
	}
	return implPath + "_test.go"
}

// TestFileInfo is the package-and-imports projection used when a
// generator emits a *_test.go alongside its primary output. The test
// package name is derived from the source per [Config.TestPackageStyle];
// the generator's import block needs the source package added explicitly
// so the test file can reference it.
type TestFileInfo struct {
	// PackageName is the test file's package declaration:
	//   "external" → "<pkg>_test"
	//   "internal" → "<pkg>"
	PackageName string

	// Imports carries the imports the test file needs in addition to
	// the original source's imports.
	Imports []Import

	// GenQualifier is the package qualifier consumers use to reference
	// types from the generator's primary output, e.g. "storetest." for
	// external tests in package storetest_test, "" for internal.
	GenQualifier string
}

// BuildTestFileInfo computes the [TestFileInfo] for a generator that
// emits a *_test.go alongside its primary output. genImportPath is the
// import path of the primary output's containing package (computed via
// [OutputImportPath]).
func BuildTestFileInfo(srcPkgName string, srcImports []Import, cfg Config, genImportPath string) TestFileInfo {
	switch cfg.TestPackageStyle {
	case TestPackageStyleInternal:
		return TestFileInfo{
			PackageName:  srcPkgName,
			Imports:      append([]Import{}, srcImports...),
			GenQualifier: "",
		}
	default:
		// External (default).
		alias := lastPathSegment(genImportPath)
		newImports := append([]Import{}, srcImports...)
		newImports = append(newImports, Import{Path: genImportPath})
		return TestFileInfo{
			PackageName:  srcPkgName + "_test",
			Imports:      newImports,
			GenQualifier: alias + ".",
		}
	}
}

// OutputCtx is the bundle of derived values every generator needs
// before rendering: where the output lands (PackageName), how to
// reference source-package symbols from there (ImportPath +
// Qualifier + Tracker), and the import path of the output file
// itself (OutputImportPath). [BuildOutputCtx] computes them in one
// call, sparing each generator the same dance.
type OutputCtx struct {
	// PackageName is the `package <name>` declaration for the output
	// file. Honors [Options.OutputPackage] (CLI -p override).
	PackageName string

	// ImportPath is the source package's import path, set when the
	// output lands in a different package and needs to import it.
	// Empty when output is same-package (no qualifier needed).
	ImportPath string

	// Qualifier is the dotted prefix used to reference source-package
	// symbols from the output ("basic." or ""). Pre-formatted with the
	// trailing dot so templates can write `{{.Qualifier}}{{.Name}}`
	// without per-template prefix logic.
	Qualifier string

	// OutputImportPath is the import path of the output file itself —
	// useful for downstream generators that emit a sibling test file.
	OutputImportPath string

	// Tracker is an [ImportTracker] seeded with the source package
	// import (when needed). Generators add further imports as they
	// render sample values for typed fields.
	Tracker *ImportTracker
}

// BuildOutputCtx computes the derived output context for a generator
// run: package name, source-import + qualifier (when needed),
// output's own import path, and a primed [ImportTracker]. Returns
// an error only when [OutputImportPath] fails (missing module info,
// unresolvable path).
//
// Every generator calls this as the first step after Analyze input
// is gathered — sparing each one the OutputPackage / DerivePackageName
// / OutputImportPath / needsImport / Tracker boilerplate.
func BuildOutputCtx(pkg *Package, cfg Config, opts Options) (*OutputCtx, error) {
	srcPkgName := pkg.Name()
	if opts.OutputPackage != "" {
		srcPkgName = opts.OutputPackage
	}
	pkgName := DerivePackageName(opts.Output, srcPkgName, cfg)

	outputImportPath, err := OutputImportPath(opts.Output, pkg, opts)
	if err != nil {
		return nil, err
	}

	var importPath, qualifier string
	if outputImportPath != pkg.Path() || pkgName != pkg.Name() {
		importPath = pkg.Path()
		qualifier = pkg.Name() + "."
	}

	tracker := NewImportTracker(outputImportPath)
	if importPath != "" {
		tracker.AddPath(importPath)
	}

	return &OutputCtx{
		PackageName:      pkgName,
		ImportPath:       importPath,
		Qualifier:        qualifier,
		OutputImportPath: outputImportPath,
		Tracker:          tracker,
	}, nil
}

// DerivePackageName picks the `package <name>` declaration for a
// generated file based on its output path, the source package name,
// and config.
//
// Rules:
//
//   - Output is a *_test.go file in the source dir:
//     "external" style → "<pkg>_test"; "internal" style → "<pkg>".
//   - Output is in the source dir (any non-test name) → source pkg.
//   - Output is in a subdirectory → that directory's basename.
func DerivePackageName(outputPath, sourcePkgName string, cfg Config) string {
	return derivePackageName(outputPath, sourcePkgName, cfg, Options{})
}

// derivePackageName is the CLI-aware variant of [DerivePackageName].
// When [Options.OutputPackage] is set (the -p flag has loaded a
// remote package), the override replaces the source pkg name — output
// lands in the CWD's package, not the remote one.
func derivePackageName(outputPath, sourcePkgName string, cfg Config, opts Options) string {
	if opts.OutputPackage != "" {
		sourcePkgName = opts.OutputPackage
	}

	dir := filepath.Dir(outputPath)
	base := filepath.Base(outputPath)

	isTestFile := strings.HasSuffix(base, "_test.go")
	isSameDir := dir == "." || dir == ""

	switch {
	case isTestFile && isSameDir:
		if cfg.TestPackageStyle == TestPackageStyleInternal {
			return sourcePkgName
		}
		return sourcePkgName + "_" + cfg.TestPackageSuffix
	case isSameDir:
		return sourcePkgName
	default:
		return filepath.Base(dir)
	}
}

// lastPathSegment returns the last "/"-delimited segment of an import
// path. Used as a default alias.
func lastPathSegment(path string) string {
	if path == "" {
		return ""
	}
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}
