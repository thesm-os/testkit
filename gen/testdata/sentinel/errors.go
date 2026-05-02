// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package sentinel is a test fixture for the sentinel generator.
package sentinel

import "errors"

// Properly prefixed sentinels.
var (
	// ErrNotFound is returned when an item is not found.
	ErrNotFound = errors.New("sentinel: not found")

	// ErrConflict is returned on duplicate key.
	ErrConflict = errors.New("sentinel: conflict")

	// ErrTimeout is returned when an operation times out.
	ErrTimeout = errors.New("sentinel: timeout")
)

// ErrForbidden is returned when access is denied.
var ErrForbidden = errors.New("sentinel: forbidden")

// unexported — should be ignored by the generator.
var errInternal = errors.New("sentinel: internal")
