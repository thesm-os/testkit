// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import (
	"fmt"
	"go/token"
)

// TypeKind constrains what kind of named type a generator expects.
type TypeKind int

const (
	// KindInterface requires the type to be an interface.
	KindInterface TypeKind = iota
	// KindStruct requires the type to be a struct.
	KindStruct
	// KindAny accepts any named type (sentinel, enum generators).
	KindAny
)

// Error is a positioned error returned by the generator engine. When
// Pos is valid, [Error.Error] formats as "file:line: message".
type Error struct {
	Pos     token.Position // file:line:col, zero value if not applicable
	Message string
	Cause   error // wrapped underlying error, if any
}

// Error formats the error with position information when available.
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

// Unwrap returns the underlying cause for [errors.Is] / [errors.As] chains.
func (e *Error) Unwrap() error { return e.Cause }

// Errorf creates a positioned error with a formatted message.
func Errorf(pos token.Position, format string, args ...any) *Error {
	return &Error{Pos: pos, Message: fmt.Sprintf(format, args...)}
}

// WrapErr creates a positioned error wrapping an underlying cause.
func WrapErr(pos token.Position, cause error, format string, args ...any) *Error {
	return &Error{Pos: pos, Message: fmt.Sprintf(format, args...), Cause: cause}
}
