// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator

import (
	"fmt"
	"go/types"
	"path"
	"sort"
	"strconv"
	"sync"
)

// ImportTracker collects import paths referenced by rendered code and
// assigns aliases when names collide. Generators thread an ImportTracker
// through every type-rendering call so the emitted file's import block
// is complete and unique.
//
// Aliases are assigned deterministically: when two packages have the
// same basename, the first to register keeps the bare name, subsequent
// registrations get suffixed aliases ("model", "model2", ...).
//
// Goroutine-safe: AddPath / Add / Qualifier / Imports may be called
// concurrently. The internal maps are guarded by an embedded mutex so
// shared trackers (e.g. one tracker passed to several parallel
// `go/types.TypeString` calls) don't race on read-modify-write of the
// alias-collision counter.
type ImportTracker struct {
	mu       sync.Mutex
	localPkg string            // import path of the package being generated
	imports  map[string]string // path → alias (alias may be empty if name == basename)
	names    map[string]int    // basename → count, for collision detection
}

// NewImportTracker creates a tracker scoped to the package being
// generated. References to localPkg are emitted unqualified.
func NewImportTracker(localPkg string) *ImportTracker {
	return &ImportTracker{
		localPkg: localPkg,
		imports:  make(map[string]string),
		names:    make(map[string]int),
	}
}

// Add registers a [types.Package] reference and returns the alias to use
// in generated code. Returns empty string when the package matches the
// tracker's localPkg (no qualifier needed).
//
// Standard library and third-party packages are tracked identically.
// goimports-format applies after rendering and may reorder the final
// import block.
func (t *ImportTracker) Add(pkg *types.Package) string {
	if pkg == nil {
		return ""
	}
	return t.AddPath(pkg.Path())
}

// AddPath registers an import by path string. Use this when the path is
// known but no [*types.Package] is in hand (e.g. injecting "testing" or
// "context" into a template).
func (t *ImportTracker) AddPath(pkgPath string) string {
	if pkgPath == "" || pkgPath == t.localPkg {
		return ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if existing, ok := t.imports[pkgPath]; ok {
		if existing != "" {
			return existing
		}
		return path.Base(pkgPath)
	}

	base := path.Base(pkgPath)
	count := t.names[base]
	t.names[base] = count + 1

	alias := ""
	if count > 0 {
		alias = fmt.Sprintf("%s%d", base, count+1)
	}
	t.imports[pkgPath] = alias

	if alias != "" {
		return alias
	}
	return base
}

// Imports returns the registered imports sorted by path. Used by
// templates to render the import block.
func (t *ImportTracker) Imports() []Import {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]Import, 0, len(t.imports))
	for p, alias := range t.imports {
		out = append(out, Import{Alias: alias, Path: p})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// LocalPkg returns the local package import path (the package being
// generated). Useful for templates that need to recognize self-references.
func (t *ImportTracker) LocalPkg() string { return t.localPkg }

// Qualifier returns a [types.Qualifier] that uses this tracker's
// import rules. Pass to [types.TypeString] for rendering arbitrary
// types.
func (t *ImportTracker) Qualifier() types.Qualifier {
	return func(pkg *types.Package) string {
		if pkg == nil {
			return ""
		}
		if pkg.Path() == t.localPkg {
			return ""
		}
		return t.AddPath(pkg.Path())
	}
}

// Import is one line of an import block.
type Import struct {
	// Alias is the import alias, or empty when the alias matches the
	// path's basename (the common case; goimports drops it).
	Alias string

	// Path is the full import path.
	Path string
}

// String renders the import as a single Go-source line ("alias \"path\"").
func (i Import) String() string {
	if i.Alias == "" {
		return strconv.Quote(i.Path)
	}
	return i.Alias + " " + strconv.Quote(i.Path)
}
