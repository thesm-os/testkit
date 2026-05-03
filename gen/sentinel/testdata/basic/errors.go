// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

import (
	"errors"
	"fmt"
)

//go:generate testkit sentinel -o errors.gen_test.go

// ErrNotFound is returned when an item is not found.
var ErrNotFound = errors.New("basic: not found")

// ErrConflict is returned on duplicate key.
var ErrConflict = errors.New("basic: conflict")

// ErrForbidden is returned when access is denied.
var ErrForbidden = errors.New("basic: forbidden")

// ValidationError is a custom error type with fields.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("basic: validation: %s: %s", e.Field, e.Message)
}

// NotFoundError has a custom Is method for matching.
type NotFoundError struct {
	Entity string
}

// Error implements the error interface.
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("basic: %s not found", e.Entity)
}

// Is implements custom error matching — any NotFoundError matches.
func (e *NotFoundError) Is(target error) bool {
	_, ok := target.(*NotFoundError)
	return ok
}

// WrappedError wraps an underlying error.
type WrappedError struct {
	Msg   string
	Cause error
}

// Error implements the error interface.
func (e *WrappedError) Error() string {
	return fmt.Sprintf("basic: %s: %v", e.Msg, e.Cause)
}

// Unwrap returns the underlying error.
func (e *WrappedError) Unwrap() error {
	return e.Cause
}
