// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

//go:generate testkit enum -o priority.gen_test.go Priority

// Priority is a bare enum (no String, no Parse, no marshalers).
// Used by the enum generator's tests to exercise the path where
// only exhaustiveness, distinctness, zero-value, and wire-compat
// subtests fire — every method-gated subtest is omitted.
type Priority int

// Priority values without inline comments — the enum generator
// falls back to the prefix-stripped const name when no comment is
// present.
const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
)
