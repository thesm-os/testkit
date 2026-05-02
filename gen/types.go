// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import (
	"go/constant"
	"go/token"
	"go/types"
)

// InterfaceInfo holds a named interface with its methods, type
// parameters, and doc comment. Methods are flattened — embedded
// interface methods are included, sorted by name.
type InterfaceInfo struct {
	Name       string
	Type       *types.Interface
	Methods    []MethodInfo
	TypeParams []TypeParamInfo
	Doc        string
	Pos        token.Position
}

// MethodInfo holds a single method with its signature, doc comment,
// and any //testkit: directives attached to it.
type MethodInfo struct {
	Name       string
	Signature  *types.Signature
	Doc        string
	Directives []Directive
	Pos        token.Position
}

// Directive is a parsed //testkit: annotation on a type or method.
//
//	//testkit:errors ErrNotFound ErrConflict
//
// Produces Directive{Name: "errors", Args: ["ErrNotFound", "ErrConflict"]}.
type Directive struct {
	Name string
	Args []string
}

// StructInfo holds a named struct with its fields, type parameters,
// and doc comment. Fields are in declaration order.
type StructInfo struct {
	Name       string
	Type       *types.Struct
	Fields     []FieldInfo
	TypeParams []TypeParamInfo
	Doc        string
	Pos        token.Position
}

// FieldInfo holds a single struct field.
type FieldInfo struct {
	Name     string
	Type     types.Type
	Exported bool
	Tag      string
}

// TypeParamInfo holds a generic type parameter with its constraint.
type TypeParamInfo struct {
	Name       string
	Constraint types.Type
}

// ConstInfo holds a named constant — used by the enum generator to
// find iota-based const blocks.
type ConstInfo struct {
	Name  string
	Type  types.Type
	Value constant.Value
	Doc   string
	Pos   token.Position
}

// VarInfo holds a named package-level variable — used by the sentinel
// generator to find exported Err* variables.
type VarInfo struct {
	Name string
	Type types.Type
	Doc  string
	Pos  token.Position
}

// GenerateDirective is a parsed //go:generate testkit line from a
// source file. Used for cross-package linking — finding where
// generated code for a type lives.
type GenerateDirective struct {
	Generator string   // "stub", "builder", "recording", etc.
	Output    string   // -o flag value, or "" for convention default
	Types     []string // type arguments
	File      string   // source file containing the directive
	Line      int
}
