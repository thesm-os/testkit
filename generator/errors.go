// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator

import (
	"fmt"
	"go/token"
)

// noPos is used internally for errors that have no source position.
var noPos = token.Position{}

// Error is a positioned error returned by the generator pipeline.
// When Pos is valid, Error() formats as "file:line:col: message".
//
// Errors flow through the pipeline carrying source positions so the
// CLI can emit IDE-navigable error messages. Generators should always
// prefer Errorf and WrapErr over fmt.Errorf when they have a position.
type Error struct {
	Pos     token.Position
	Message string
	Cause   error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Pos.IsValid() {
		if e.Cause != nil {
			return fmt.Sprintf("%s: %s: %v", e.Pos, e.Message, e.Cause)
		}
		return fmt.Sprintf("%s: %s", e.Pos, e.Message)
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying cause for errors.Is/errors.As traversal.
func (e *Error) Unwrap() error { return e.Cause }

// Errorf creates a positioned [Error]. Use noPos when no source position
// applies (e.g., registration errors).
func Errorf(pos token.Position, format string, args ...any) *Error {
	return &Error{Pos: pos, Message: fmt.Sprintf(format, args...)}
}

// WrapErr creates a positioned [Error] wrapping a cause. Use this when
// rethrowing an underlying error with additional context.
func WrapErr(pos token.Position, cause error, format string, args ...any) *Error {
	return &Error{Pos: pos, Message: fmt.Sprintf(format, args...), Cause: cause}
}

// TypeKind constrains what kind of named type a generator expects in its
// args. Used by [ValidateTypes] to fail early with clear diagnostics.
type TypeKind int

// Recognized type kinds.
const (
	// KindAny accepts any named type. Generators that scan whole packages
	// (sentinel) use this.
	KindAny TypeKind = iota

	// KindInterface accepts only interface types. Used by stub, suite,
	// bench, model.
	KindInterface

	// KindStruct accepts only struct types. Used by builder.
	KindStruct

	// KindNamedType accepts any defined type — typically used by enum
	// to accept the underlying integer type of an iota block.
	KindNamedType
)

// String returns a short label for the kind, used in error messages.
func (k TypeKind) String() string {
	switch k {
	case KindInterface:
		return "interface"
	case KindStruct:
		return "struct"
	case KindNamedType:
		return "named type"
	default:
		return "type"
	}
}
