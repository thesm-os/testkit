// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator

import (
	"go/constant"
	"go/token"
	"go/types"

	"go.thesmos.sh/testkit/generator/directive"
)

// MethodInfo describes a method on an interface or concrete type. The
// generator pipeline operates on slices of MethodInfo regardless of
// whether they came from interfaces or concrete types — the loader
// produces them from either source.
type MethodInfo struct {
	// Name is the method's identifier ("Get", "Put").
	Name string

	// Signature is the resolved function signature from go/types.
	Signature *types.Signature

	// Doc is the method's leading doc comment (no leading "//").
	Doc string

	// Directives are the //testkit: annotations on this method.
	Directives []directive.Directive

	// Pos is the source position of the method declaration.
	Pos token.Position
}

// InterfaceInfo describes a Go interface. The Methods slice is sorted
// by name for deterministic output and includes promoted methods from
// embedded interfaces.
type InterfaceInfo struct {
	Name       string
	Type       *types.Interface
	Methods    []MethodInfo // sorted by name; embedded methods promoted
	TypeParams []TypeParamInfo
	Doc        string
	Directives []directive.Directive
	Pos        token.Position
}

// StructInfo describes a Go struct. The Fields slice preserves
// declaration order (relevant for builder generation).
type StructInfo struct {
	Name       string
	Type       *types.Struct
	Fields     []FieldInfo // declaration order
	TypeParams []TypeParamInfo
	Doc        string
	Directives []directive.Directive
	Pos        token.Position
}

// FieldInfo describes one struct field. Used for builder generation
// and for reading struct-level field directives (e.g. //testkit:default).
type FieldInfo struct {
	// Name is the field identifier. Unexported fields are emitted by
	// the loader for completeness; consumers filter on Exported when
	// they only want the public API.
	Name string

	// Type is the resolved go/types Type for the field.
	Type types.Type

	// Exported reports whether the field is exported.
	Exported bool

	// Tag is the raw struct tag value, or empty.
	Tag string

	// Directives are //testkit: annotations on the field's doc comment.
	Directives []directive.Directive

	// InlineComment is the trailing line comment after the field
	// declaration (e.g. // optional default), or empty. Used by some
	// directives (notably default) that allow inline argument forms.
	InlineComment string
}

// TypeParamInfo describes one type parameter on a generic interface or
// struct. Generators use this to preserve type parameters end-to-end in
// emitted code.
type TypeParamInfo struct {
	Name       string
	Constraint types.Type
}

// VarInfo describes a package-level variable. Used by the sentinel
// generator (Err* vars) and by directive enrichers that resolve named
// values to packages.
type VarInfo struct {
	Name string
	Type types.Type
	Doc  string
	Pos  token.Position
}

// ConstInfo describes a package-level constant. Used by the enum
// generator.
type ConstInfo struct {
	Name  string
	Type  types.Type
	Value constant.Value

	// Doc is the doc comment attached to the const declaration
	// (the "// Foo is the X" line above the const, when present).
	Doc string

	// Comment is the inline comment trailing the const declaration
	// (the "// Pending" suffix on `StatusPending Status = iota // Pending`),
	// trimmed of the leading "// ". Empty when no inline comment.
	// Used by the enum generator to derive expected stringer output.
	Comment string

	Pos token.Position
}

// FieldData is the rendered form of one struct field or function
// parameter, suitable for direct use in templates. Unlike FieldInfo,
// FieldData carries the type as a pre-rendered string (qualified via
// an [ImportTracker]) so templates don't manipulate go/types.
//
// Used by stub call types, builder fields, and recording call types.
type FieldData struct {
	// FieldName is the Go identifier with first letter capitalized,
	// initialism-promoted.
	//   "id"          → "ID"
	//   "user_id"     → "UserID"
	//   "httpRequest" → "HTTPRequest"
	FieldName string

	// TypeStr is the type rendered as Go source, qualified via
	// the ImportTracker that produced this FieldData.
	//   "context.Context"
	//   "store.Item"
	//   "[]string"
	TypeStr string

	// ZeroValue is a Go expression that evaluates to the zero value
	// of TypeStr. Used by stub fault paths and by buildTestData.
	//   "" for strings, "0" for ints, "nil" for slices/maps/pointers,
	//   "Item{}" for structs, etc.
	ZeroValue string

	// IsError is true when this field carries the error return.
	// Set by [BuildResultFields] for the last result if it's of type
	// error.
	IsError bool
}

// IterSeqInfo describes an iter.Seq[T] or iter.Seq2[V, error] return
// type. Set by [DetectIter] (see method.go) and consumed by stream
// shape detection and stub Yields helper generation.
type IterSeqInfo struct {
	// IsSeq is true when the return type is iter.Seq[T].
	IsSeq bool

	// IsSeq2 is true when the return type is iter.Seq2[T, U].
	IsSeq2 bool

	// Seq2Error is true when IsSeq2 and the second type is error.
	Seq2Error bool

	// ValType is the rendered string for T (Seq) or V (Seq2).
	ValType string

	// ErrType is the rendered string for U in Seq2 (typically "error").
	// Empty when not Seq2 or when Seq2 doesn't carry error.
	ErrType string
}
