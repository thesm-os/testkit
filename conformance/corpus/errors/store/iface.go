// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package store is the errors-kind fixture: the input the sentinel generator
// reads.
//
// The directive sits at package scope rather than on the variables. sentinel
// declares On(node.KindPackage), so what a generator *reads* and where it is
// opted in are different things — a fixture carrying a var-level directive
// would never be picked up.
//
//testkit:sentinel prefix=store
//testkit:sentinel-no-overlap-with go.thesmos.sh/testkit/conformance/corpus/errors/neighbour
package store

import (
	"errors"
	"fmt"
)

// The sentinel set. Every message carries the package prefix, none is a
// prefix of another, and none satisfies errors.Is against a sibling — which
// are the three properties the generated checks assert.
var (
	// ErrNotFound reports a key that was never written.
	ErrNotFound = errors.New("store: not found")

	// ErrConflict reports a write losing a race.
	ErrConflict = errors.New("store: conflict")

	// ErrClosed reports use after teardown.
	ErrClosed = errors.New("store: closed")
)

// NotFoundError is a custom error carrying a field. The generated round-trip
// check formats it, parses the field back out, and compares — so the field has
// to survive the format string.
type NotFoundError struct {
	Key string
}

// Error implements error.
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("store: key %q not found", e.Key)
}

// Is reports NotFoundError as a kind of [ErrNotFound], which is the optional
// method the generator detects and exercises.
func (*NotFoundError) Is(target error) bool { return target == ErrNotFound }

// WrappedError wraps a cause, so the generated unwrap-chain check has
// something to walk.
type WrappedError struct {
	Op    string
	Cause error
}

// Error implements error.
func (e *WrappedError) Error() string {
	return fmt.Sprintf("store: %s: %v", e.Op, e.Cause)
}

// Unwrap exposes the cause.
func (e *WrappedError) Unwrap() error { return e.Cause }
