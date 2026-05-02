// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import (
	"go/types"
	"path/filepath"
	"strings"
)

const (
	testFileSuffix = "_test.go"
	testPkgSuffix  = "_test"
	currentDir     = "."
)

// Options holds per-invocation settings for a generator.
type Options struct {
	Output     string // -o flag value, empty for convention default
	Check      bool   // dry-run mode — compare but don't write
	Verbose    bool
	BuildTag   string // e.g. "integration" for //go:build tag
	WorkDir    string // directory //go:generate runs in
	SourceFile string // $GOFILE — the file containing the //go:generate directive
}

// Result holds the output files from a single generator invocation.
type Result struct {
	Files []OutputFile
}

// OutputFile is a single generated file with its path and content.
type OutputFile struct {
	Path    string // relative to WorkDir
	Content []byte // formatted, with header
}

// DerivePackageName computes the Go package name for a generated file
// based on its output path, the source package name, and config.
//
// Rules:
//   - Output in source package dir → source package name.
//   - Output in <pkg><suffix>/ dir → <pkg><suffix>.
//   - Output ending in _test.go → source package name + "_test".
//   - TestPackageStyle "internal" → always source package name.
func DerivePackageName(outputPath, sourcePkgName string, cfg Config) string {
	dir := filepath.Dir(outputPath)
	base := filepath.Base(outputPath)

	// _test.go files in the source directory.
	if dir == currentDir && strings.HasSuffix(base, testFileSuffix) {
		if cfg.TestPackageStyle == TestPackageStyleInternal {
			return sourcePkgName
		}
		return sourcePkgName + testPkgSuffix
	}

	// Same directory as source.
	if dir == currentDir {
		return sourcePkgName
	}

	// Output in a subdirectory — use the directory name as package name.
	return filepath.Base(dir)
}

// OutputImportPath computes the Go import path for a generated file
// given its output path relative to the source package and the
// source package's module and import path.
func OutputImportPath(outputPath string, pkg *Package) (string, error) {
	dir := filepath.Dir(outputPath)
	if dir == currentDir {
		return pkg.Pkg.Path(), nil
	}
	return pkg.Pkg.Path() + "/" + filepath.ToSlash(dir), nil
}

// ValidateTypes checks that all named types exist in the package and
// are of the expected kind. Returns a slice of positioned errors for
// any failures.
func ValidateTypes(pkg *Package, names []string, kind TypeKind) []*Error {
	var errs []*Error
	scope := pkg.Pkg.Scope()
	for _, name := range names {
		obj := scope.Lookup(name)
		if obj == nil {
			errs = append(errs, Errorf(
				pkg.Fset.Position(pkg.Syntax[0].Package),
				"type %q not found in package %s", name, pkg.Pkg.Name(),
			))
			continue
		}
		if kind == KindAny {
			continue
		}
		named, ok := obj.Type().(*types.Named)
		if !ok {
			errs = append(errs, Errorf(
				pkg.Fset.Position(obj.Pos()),
				"%q is not a named type", name,
			))
			continue
		}
		switch kind {
		case KindInterface:
			if _, ok := named.Underlying().(*types.Interface); !ok {
				errs = append(errs, Errorf(
					pkg.Fset.Position(obj.Pos()),
					"%q is a %s, not an interface", name, typeKindString(named.Underlying()),
				))
			}
		case KindStruct:
			if _, ok := named.Underlying().(*types.Struct); !ok {
				errs = append(errs, Errorf(
					pkg.Fset.Position(obj.Pos()),
					"%q is a %s, not a struct", name, typeKindString(named.Underlying()),
				))
			}
		case KindAny:
			// Already handled above the switch.
		}
	}
	return errs
}

func typeKindString(typ types.Type) string {
	switch typ.(type) {
	case *types.Interface:
		return "interface"
	case *types.Struct:
		return "struct"
	case *types.Basic:
		return "basic type"
	case *types.Slice:
		return "slice"
	case *types.Map:
		return "map"
	default:
		return "type"
	}
}
