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
	TypeParamDecl       string      // "[T any]" or "" for non-generic
	TypeParamArgs       string      // "[T]" or "" for non-generic
	IsGeneric           bool        // true if struct has type parameters
	TestTypeArgs        string      // "[string]" — concrete types for generated tests
	TestQualifiedType   string      // "generics.Container[string]" — for test assertions
}

// FieldData holds one exported field for a With* setter.
type FieldData struct {
	Name          string // "ID"
	TypeStr       string // "string"
	SampleValue   string // `"test-id"` — non-zero sample for tests
	IsSlice       bool   // true if the field type is a slice (variadic setter)
	ElemTypeStr   string // "string" for []string (only set when IsSlice)
	IsMap         bool   // true if the field type is a map
	MapKeyTypeStr string // "string" for map[string]int (only set when IsMap)
	MapValTypeStr string // "int" for map[string]int (only set when IsMap)
	MapKeySample  string // `"test-key"` — sample key for tests
	MapValSample  string // `"test-val"` — sample value for tests
	IsBytes       bool   // true if the field type is []byte
	IsStruct      bool   // true if the field type is a struct (named or anonymous)
	IsPointer     bool   // true if the field type is a pointer
	DefaultValue  string // from //testkit:default "value" or "" if none
	TestTypeStr   string // concrete type for generic test instantiation ("string" for "T")
	TestSample    string // sample value using concrete type (`"test-name"` for string)
}

// FirstComparableField returns the first field that supports !=
// comparison and has a non-zero sample value (basic types: string,
// int, bool). Falls back to the first field if none qualify.
func (s *StructData) FirstComparableField() FieldData {
	for _, f := range s.Fields {
		if !f.IsSlice && !f.IsMap && !f.IsBytes && !f.IsStruct && !f.IsPointer {
			return f
		}
	}
	if len(s.Fields) > 0 {
		return s.Fields[0]
	}
	return FieldData{}
}
