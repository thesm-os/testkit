// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import (
	"go/types"
	"path"
	"sort"
	"strconv"
)

// Import represents a single import in a generated file.
type Import struct {
	Alias string // empty when package name matches last path element
	Path  string
}

// ImportTracker collects import paths needed by generated code and
// assigns short aliases when package names collide. Thread-unsafe —
// use one tracker per generated file.
type ImportTracker struct {
	localPkg string
	imports  map[string]string // path → alias (or package name)
	names    map[string]int    // base name → count for collision detection
}

// NewImportTracker returns an [ImportTracker] for a generated file in
// the package with the given import path. Imports of localPkgPath are
// omitted from the output (same package, no qualifier needed).
func NewImportTracker(localPkgPath string) *ImportTracker {
	return &ImportTracker{
		localPkg: localPkgPath,
		imports:  make(map[string]string),
		names:    make(map[string]int),
	}
}

// Add registers a package import and returns the qualifier to use in
// generated code. Returns empty string if pkg is the local package.
func (t *ImportTracker) Add(pkg *types.Package) string {
	if pkg == nil {
		return ""
	}
	pkgPath := pkg.Path()
	if pkgPath == t.localPkg {
		return ""
	}
	if alias, ok := t.imports[pkgPath]; ok {
		return alias
	}
	name := pkg.Name()
	return t.addWithName(pkgPath, name)
}

// AddPath registers an import by path and returns the qualifier. The
// package name is derived from the last element of the path.
func (t *ImportTracker) AddPath(pkgPath string) string {
	if pkgPath == t.localPkg {
		return ""
	}
	if alias, ok := t.imports[pkgPath]; ok {
		return alias
	}
	name := path.Base(pkgPath)
	return t.addWithName(pkgPath, name)
}

// Imports returns all collected imports sorted by path. The local
// package is excluded.
func (t *ImportTracker) Imports() []Import {
	result := make([]Import, 0, len(t.imports))
	for pkgPath, alias := range t.imports {
		imp := Import{Path: pkgPath}
		// Only set alias if it differs from the last path element.
		baseName := path.Base(pkgPath)
		if alias != baseName {
			imp.Alias = alias
		}
		result = append(result, imp)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

// Qualifier returns a [types.Qualifier] function suitable for
// [types.TypeString]. It calls [ImportTracker.Add] for each
// referenced package.
func (t *ImportTracker) Qualifier() types.Qualifier {
	return func(pkg *types.Package) string {
		return t.Add(pkg)
	}
}

func (t *ImportTracker) addWithName(pkgPath, name string) string {
	count := t.names[name]
	t.names[name] = count + 1
	alias := name
	if count > 0 {
		alias = name + strconv.Itoa(count+1)
	}
	t.imports[pkgPath] = alias
	return alias
}
