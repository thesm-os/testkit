// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package enum implements the enum generator for testkit. It scans const
// blocks of named types and generates exhaustiveness, distinctness, and
// stringer tests.
package enum

import (
	"go/constant"
	"go/types"

	"go.thesmos.sh/testkit/gen"
)

// Data is the top-level template data for an enum generation run.
type Data struct {
	PackageName  string
	ImportPath   string // source package import path (empty if same package)
	PkgQualifier string // "store" or "" for same package
	Enums        []TypeData
}

// HasContent reports whether there are any enums to test.
func (d *Data) HasContent() bool {
	return len(d.Enums) > 0
}

// HasStringer reports whether any enum has a String() method.
func (d *Data) HasStringer() bool {
	for _, e := range d.Enums {
		if e.HasString {
			return true
		}
	}
	return false
}

// TypeData holds one enum type and its constant values.
type TypeData struct {
	TypeName  string  // "Status"
	Values    []Value // sorted by name
	MaxValue  int64   // highest iota value for boundary test
	HasString bool    // true if the type has a String() string method
}

// Value holds one constant of an enum type.
type Value struct {
	Name        string // "StatusPending"
	ExpectedStr string // "Pending" — inline comment or derived name
	IntValue    int64
}

// Analyze builds Data by scanning the package for const blocks of
// the given named types.
func Analyze(
	pkg *gen.Package,
	args []string,
	cfg gen.Config,
	opts gen.Options,
) (*Data, error) {
	pkgName := gen.DerivePackageName(opts.Output, pkg.Pkg.Name(), cfg)

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

	var enums []TypeData

	for _, typeName := range args {
		consts := pkg.ConstsOfType(typeName)
		if len(consts) == 0 {
			continue
		}

		ed := TypeData{
			TypeName:  typeName,
			HasString: hasStringMethod(pkg, typeName),
		}

		var maxVal int64
		for _, c := range consts {
			intVal := int64(0)
			if c.Value.Kind() == constant.Int {
				v, _ := constant.Int64Val(c.Value)
				intVal = v
				if v > maxVal {
					maxVal = v
				}
			}

			// Derive expected string from inline comment or by stripping
			// the type name prefix.
			expected := c.Comment
			if expected == "" {
				expected = stripPrefix(c.Name, typeName)
			}

			ed.Values = append(ed.Values, Value{
				Name:        c.Name,
				ExpectedStr: expected,
				IntValue:    intVal,
			})
		}
		ed.MaxValue = maxVal
		enums = append(enums, ed)
	}

	return &Data{
		PackageName:  pkgName,
		ImportPath:   importPath,
		PkgQualifier: qualifier,
		Enums:        enums,
	}, nil
}

// hasStringMethod checks if a type has a String() string method.
func hasStringMethod(pkg *gen.Package, typeName string) bool {
	obj := pkg.Pkg.Scope().Lookup(typeName)
	if obj == nil {
		return false
	}
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return false
	}
	mset := types.NewMethodSet(named)
	for sel := range mset.Methods() {
		fn, ok := sel.Obj().(*types.Func)
		if !ok || fn.Name() != "String" {
			continue
		}
		sig := fn.Type().(*types.Signature)
		if sig.Params().Len() == 0 && sig.Results().Len() == 1 {
			return true
		}
	}
	return false
}

// stripPrefix removes the type name prefix from a const name.
// "StatusPending" with type "Status" -> "Pending".
func stripPrefix(constName, typeName string) string {
	hasPrefix := len(constName) > len(typeName) &&
		constName[:len(typeName)] == typeName
	if hasPrefix {
		return constName[len(typeName):]
	}
	return constName
}
