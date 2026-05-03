// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package builder implements the builder generator for testkit. It produces
// fluent test fixture builders from Go struct definitions.
package builder

import "go.thesmos.sh/testkit/gen"

// Data is the top-level template data for a builder generation run.
type Data struct {
	PackageName  string
	Imports      []gen.Import
	Structs      []StructData
	GenQualifier string // package qualifier for test file (e.g. "storetest." or "")
}

// StructData holds one struct being built.
type StructData struct {
	Name                string      // "Item"
	BuilderName         string      // "ItemBuilder"
	QualifiedType       string      // "store.Item"
	Fields              []FieldData // exported fields, sorted alphabetically
	HasUnexportedFields bool        // true if struct has unexported fields
	HasDefaults         bool        // true if <Type>Defaults() exists in output package
	DefaultsFunc        string      // "ItemDefaults" or "" if no defaults
	HasFieldDefaults    bool        // true if any field has //testkit:default
}

// FieldData holds one exported field for a With* setter.
type FieldData struct {
	Name         string // "ID"
	TypeStr      string // "string"
	SampleValue  string // `"test-id"` — non-zero sample for tests
	IsSlice      bool   // true if the field type is a slice (variadic setter)
	ElemTypeStr  string // "string" for []string (only set when IsSlice)
	DefaultValue string // from //testkit:default "value" or "" if none
}

// FirstComparableField returns the first non-slice field, or the
// first field if all are slices. Used by test templates for Clone
// assertions where != must compile.
func (s *StructData) FirstComparableField() FieldData {
	for _, f := range s.Fields {
		if !f.IsSlice {
			return f
		}
	}
	if len(s.Fields) > 0 {
		return s.Fields[0]
	}
	return FieldData{}
}
