// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package gate proves the conformance corpus is complete and that the code
// generated from it is worth having.
//
// Nothing here reads a manifest of what the corpus covers. Coverage is
// measured by running eidos's own shape plugin over the corpus and collecting
// what it stamps, because a hand-maintained list drifts and, worse, passes for
// a fixture whose directive is misspelled — the folder is named correctly and
// the list agrees with the folder.
//
// # Hazards
//
// The gate depends on eidos's annotator at test time, so an upstream
// classification bug surfaces here as a corpus gap and points at the wrong
// repository. That is a deliberate trade against the alternative, where a
// stale manifest reports success. [Coverage.String] prints the stamped set so
// the cause is visible rather than inferred.
package gate

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// corpusRoot is the directory the corpus lives under, relative to the module
// root. Kept as a constant because both the walker and its test resolve
// against it.
const corpusRoot = "corpus"

// Package is one corpus fixture: a directory holding an interface, a struct,
// an enum, or an error set, plus the implementations a generated suite runs
// against.
type Package struct {
	// Dir is the path relative to the module root, e.g.
	// "corpus/iface/detector/reader".
	Dir string

	// Kind is the input kind the fixture exercises — the first path segment
	// under the corpus root: "iface", "struct", "enum", or "errors".
	Kind string

	// Axis is the classification axis for iface fixtures — "detector",
	// "contract", "mixin", "lang", or "composite". Empty for other kinds,
	// which have no axis subdivision.
	Axis string

	// Name is the leaf directory name. For detector, contract, and mixin
	// fixtures it is the canonical classification name, which is how a reader
	// finds the fixture for a given classification. The gate does not rely on
	// it: coverage comes from what the annotator stamps.
	Name string
}

// ImportPath returns the package's import path within this module.
func (p Package) ImportPath() string {
	return "go.thesmos.sh/testkit/conformance/" + filepath.ToSlash(p.Dir)
}

// Walk discovers every corpus fixture under root.
//
// A directory is a fixture when it contains at least one non-generated Go
// file. Directories that hold only generated output — the `<name>test`
// packages a generator writes beside its input — are skipped, so the walk
// returns inputs rather than outputs.
//
// Results are sorted by Dir so callers iterate deterministically; a gate that
// reported gaps in filesystem order would produce a different diff on every
// machine.
func Walk(root string) ([]Package, error) {
	base := filepath.Join(root, corpusRoot)
	var out []Package

	err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		has, hErr := hasFixtureSource(path)
		if hErr != nil {
			return hErr
		}
		if !has {
			return nil
		}
		// WalkDir hands back paths that descend from base, which descends
		// from root, so trimming the prefix is total where filepath.Rel would
		// add an error arm nothing can reach.
		rel := strings.TrimPrefix(filepath.ToSlash(path), filepath.ToSlash(root)+"/")
		out = append(out, classify(rel))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("gate: walk corpus: %w", err)
	}

	slices.SortFunc(out, func(a, b Package) int { return strings.Compare(a.Dir, b.Dir) })
	return out, nil
}

// classify splits a corpus-relative directory into its kind, axis, and name.
//
// The path shape differs by kind: iface fixtures nest one level deeper for the
// classification axis (corpus/iface/detector/reader) while the others do not
// (corpus/enum/status). Anything shallower than a leaf carries empty fields
// rather than erroring, because a partially-populated corpus is a state the
// gate reports rather than a state that stops it.
func classify(dir string) Package {
	p := Package{Dir: dir}
	seg := strings.Split(dir, "/")
	if len(seg) < 2 {
		return p
	}
	p.Kind = seg[1]
	p.Name = seg[len(seg)-1]
	if p.Kind == "iface" && len(seg) >= 4 {
		p.Axis = seg[2]
	}
	return p
}

// hasFixtureSource reports whether dir holds at least one hand-written Go
// file. Generated files carry the `.gen.go` suffix and test files are not
// fixtures, so neither counts.
func hasFixtureSource(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, fmt.Errorf("gate: read %s: %w", dir, err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		if strings.HasSuffix(name, ".gen.go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		return true, nil
	}
	return false, nil
}
