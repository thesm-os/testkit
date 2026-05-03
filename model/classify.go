// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import "fmt"

// FailureKind classifies a model-test failure for structured reporting.
type FailureKind int

const (
	// FailureStructural indicates a structural violation (chain hash
	// mismatch, ordering violation).
	FailureStructural FailureKind = iota

	// FailureSemantic indicates a SUT vs reference mismatch.
	FailureSemantic

	// FailureInvariant indicates a cross-shape law violation.
	FailureInvariant

	// FailureLiveness indicates no progress in N commands (deadlock).
	FailureLiveness
)

// String returns the failure kind name.
func (k FailureKind) String() string {
	switch k {
	case FailureStructural:
		return "structural"
	case FailureSemantic:
		return "semantic"
	case FailureInvariant:
		return "invariant"
	case FailureLiveness:
		return "liveness"
	default:
		return "unknown"
	}
}

// Failure is a classified model-test failure with optional REQ tag.
type Failure struct {
	Kind    FailureKind
	LawID   string // e.g., "AUTO-READ-AFTER-WRITE" or consumer-supplied
	REQID   string // e.g., "REQ-PKG-FOO-001", empty if untagged
	Step    int    // command index in the sequence
	Message string
	Err     error
}

// Error implements the error interface.
func (f *Failure) Error() string {
	prefix := f.Kind.String()
	if f.REQID != "" {
		prefix = f.REQID + " " + prefix
	}
	if f.LawID != "" {
		return fmt.Sprintf("[%s] %s at step %d: %s", prefix, f.LawID, f.Step, f.Message)
	}
	return fmt.Sprintf("[%s] at step %d: %s", prefix, f.Step, f.Message)
}

// Unwrap returns the underlying error.
func (f *Failure) Unwrap() error { return f.Err }
