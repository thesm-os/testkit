// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

// Status is a sample enum constant type for ConstsOfType-style tests.
type Status int

// Status values exercised by enum-fixture tests.
const (
	StatusPending Status = iota
	StatusActive
	StatusClosed
)
