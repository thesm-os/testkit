// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"fmt"

	"go.thesmos.sh/testkit/core/trace"
)

// FailureKind classifies a model-test failure for structured reporting.
type FailureKind int

const nilStr = "<nil>"

const (
	// FailureUnclassified is the zero value — indicates the action did not
	// set a Kind. The formatter renders as [unclassified].
	FailureUnclassified FailureKind = iota

	// FailureStructural indicates a structural violation (chain hash
	// mismatch, ordering violation). Linearizability mismatches are
	// classified as [FailureSemantic] — the SUT diverged from the
	// linearizable reference model.
	FailureStructural

	// FailureSemantic indicates a SUT vs reference mismatch,
	// including non-linearizable concurrent histories.
	FailureSemantic

	// FailureInvariant indicates a cross-shape law violation.
	FailureInvariant

	// FailureLiveness indicates a goroutine leak or deadlock.
	FailureLiveness
)

// String returns the failure kind name.
func (k FailureKind) String() string {
	switch k {
	case FailureUnclassified:
		return "unclassified"
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

// StepID identifies where in the execution an action ran.
type StepID struct {
	WorkerID int // -1 = sequential (no worker); >= 0 = concurrent worker ID
	Index    int // 0-based action index within worker (concurrent) or iteration (sequential)
}

// Failure is a classified model-test failure with structured context.
type Failure struct { //nolint:errname // Failure is the established name; renaming would break consumers
	Kind          FailureKind
	LawID         string // e.g., "AUTO-READ-AFTER-WRITE"; empty for action failures
	REQID         string // e.g., "REQ-PKG-FOO-001"; empty for actions and untagged laws
	StepRan       StepID // index of the action that failed
	StepReported  StepID // same as StepRan for sequential; may differ for concurrent
	Err           error  // canonical structured error
	Trace         []trace.Event
	ArtifactPaths []string

	// SUTState and RefState are snapshots captured at the failure
	// point, rendered via fmt.Sprintf("%+v", ...). Populated by the
	// runner for invariant violations so reporters can show the
	// state at violation alongside the law's diff. Empty for kinds
	// that don't carry per-step state (Structural, Liveness).
	SUTState string
	RefState string
}

// Error implements the error interface.
func (f *Failure) Error() string {
	prefix := f.Kind.String()
	if f.REQID != "" {
		prefix = f.REQID + " " + prefix
	}
	step := f.StepRan.Index
	msg := nilStr
	if f.Err != nil {
		msg = f.Err.Error()
	}
	if f.LawID != "" {
		return fmt.Sprintf("[%s] %s at step %d: %s", prefix, f.LawID, step, msg)
	}
	return fmt.Sprintf("[%s] at step %d: %s", prefix, step, msg)
}

// Unwrap returns the underlying error.
func (f *Failure) Unwrap() error { return f.Err }
