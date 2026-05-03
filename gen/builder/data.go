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
	Fields              []FieldData // exported fields only
	HasUnexportedFields bool        // true if struct has unexported fields
}

// FieldData holds one exported field for a With* setter.
type FieldData struct {
	Name        string // "ID"
	TypeStr     string // "string"
	SampleValue string // `"test-id"` — non-zero sample for tests
}
