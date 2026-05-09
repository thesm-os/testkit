// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package failure

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.thesmos.sh/testkit/trace"
)

// Position is a source position used in [Failure.Pos] for codegen-
// time directive errors and runtime errors that resolve to a Go
// source line. Filename is typically a relative path from the
// repository root; Line and Column are 1-based.
type Position struct {
	Filename string `json:"filename"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
}

// IsZero reports whether the Position is empty (no source location).
func (p Position) IsZero() bool {
	return p.Filename == "" && p.Line == 0 && p.Column == 0
}

// String formats as "<filename>:<line>:<column>" for filenames; the
// empty Position renders as "<unknown>".
func (p Position) String() string {
	if p.IsZero() {
		return "<unknown>"
	}
	return fmt.Sprintf("%s:%d:%d", p.Filename, p.Line, p.Column)
}

// Failure is the unified envelope every generator emits when an
// assertion fires. Each generator sets [Failure.Generator] to its
// name and populates [Failure.Details] with a structured payload
// specific to that layer; CI tooling consumes one shape across all
// generators.
//
//nolint:errname // Failure is the established cross-spec name; the FailureError suffix would force every consumer through a rename
type Failure struct {
	// Kind classifies the failure for routing to per-kind reporters.
	Kind Kind `json:"kind"`

	// REQID is the primary requirement traced through this failure
	// when known. The REQ-to-law coverage matrix indexes by this
	// field. Empty when no REQ tag was carried into the failing
	// observation.
	REQID string `json:"req_id,omitempty"`

	// Pos is the source position when the failure can be tied to
	// one — a directive validation error, a Go source line a
	// runtime check resolves to. Zero-valued for failures that
	// have no source location.
	Pos Position `json:"pos,omitzero"`

	// Subject is the interface or subsystem name the failure
	// pertains to ("basic.Store", "ledger.Subsystem"). Used in
	// the Error() string and in CI artifact names.
	Subject string `json:"subject,omitempty"`

	// Generator is the producing generator's name: "model" / "sim"
	// / "chaos" / "diff-rollout" / "replay". CI ingestion routes
	// per-generator detail handlers from this field.
	Generator string `json:"generator,omitempty"`

	// Seed is the run seed that produced the failure. Pinning a
	// rerun to the same seed is the regression workflow.
	Seed int64 `json:"seed,omitempty"`

	// Trace is the captured engine or per-interface trace at
	// failure time. Nil when the producer didn't capture (e.g.,
	// codegen-time validation errors that don't have a runtime
	// trace).
	Trace *trace.Trace `json:"trace,omitempty"`

	// Snapshot holds per-component or per-impl state captured at
	// failure tick. Nil or empty when no state was captured.
	Snapshot *Snapshot `json:"snapshot,omitempty"`

	// Artifacts references on-disk files dumped at failure time:
	// failfile, Porcupine HTML, classified-failure JSON, timeline
	// HTML, etc.
	Artifacts []Artifact `json:"artifacts,omitempty"`

	// Details carries generator-specific structured data that
	// doesn't fit the cross-cutting fields. The model generator
	// stores LawID and StepID here; chaos stores the load-bearing
	// fault set; diff-rollout stores the divergence path; replay
	// stores the trace-source provenance.
	Details map[string]any `json:"details,omitempty"`

	// Err is the underlying error if any. Round-trips through JSON
	// as `{"err": "<message>"}`; the message is preserved but the
	// original error type is not.
	Err error `json:"-"`

	// Time is wall-clock at failure capture, used for sorting
	// failure artifacts in CI. Not load-bearing for any decision
	// logic; pure metadata.
	Time time.Time `json:"time"`
}

// Error implements the error interface. The format prefixes the
// generator and kind, then cites REQ, Subject, and the underlying
// error message:
//
//	[model/invariant] [REQ-STORE-001] basic.Store: read-after-write violated
//	[chaos/chaos-crash] ledger.Subsystem: panic during NetworkPartition
//
// Empty fields collapse: a Failure without REQID skips the bracket;
// without Subject the colon is omitted.
func (f *Failure) Error() string {
	prefix := ""
	if f.Generator != "" {
		prefix = f.Generator + "/"
	}
	prefix += f.Kind.String()

	parts := []string{"[" + prefix + "]"}
	if f.REQID != "" {
		parts = append(parts, "["+f.REQID+"]")
	}
	if f.Subject != "" {
		parts = append(parts, f.Subject+":")
	}
	if f.Err != nil {
		parts = append(parts, f.Err.Error())
	}
	return strings.Join(parts, " ")
}

// Unwrap returns the underlying error so [errors.Is] and
// [errors.As] traverse to the cause.
func (f *Failure) Unwrap() error { return f.Err }

// New constructs a Failure with required fields set and Time
// populated from the wall clock. Optional fields (REQID, Pos,
// Subject, Trace, Snapshot, Artifacts, Details) are left at their
// zero values for the caller to populate.
func New(generator string, kind Kind, err error) *Failure {
	return &Failure{
		Generator: generator,
		Kind:      kind,
		Err:       err,
		Time:      time.Now(),
	}
}

// MarshalJSON encodes a Failure with custom handling for [Failure.Err]
// (rendered as a string) and [Failure.Kind] (rendered as a name).
// The trace, snapshot, and artifacts marshal naturally via their
// own struct tags.
func (f *Failure) MarshalJSON() ([]byte, error) {
	type alias Failure
	out := struct {
		*alias
		KindName string `json:"kind"`
		ErrMsg   string `json:"err,omitempty"`
	}{
		alias:    (*alias)(f),
		KindName: f.Kind.String(),
	}
	if f.Err != nil {
		out.ErrMsg = f.Err.Error()
	}
	b, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("failure.Failure.MarshalJSON: %w", err)
	}
	return b, nil
}

// UnmarshalJSON decodes a Failure JSON envelope, restoring the
// [Failure.Kind] from its string name and rebuilding [Failure.Err]
// as a synthesized error carrying the recorded message. The
// original error type is not preserved (Err round-trips as a string,
// not as a typed value).
func (f *Failure) UnmarshalJSON(b []byte) error {
	type alias Failure
	in := struct {
		*alias
		KindName string `json:"kind"`
		ErrMsg   string `json:"err,omitempty"`
	}{alias: (*alias)(f)}
	if err := json.Unmarshal(b, &in); err != nil {
		return fmt.Errorf("failure.Failure.UnmarshalJSON: %w", err)
	}
	if in.KindName != "" {
		k, err := ParseKind(in.KindName)
		if err != nil {
			return err
		}
		f.Kind = k
	}
	if in.ErrMsg != "" {
		f.Err = jsonError(in.ErrMsg)
	}
	return nil
}

// jsonError is a minimal error type used by [Failure.UnmarshalJSON]
// to carry an error message recovered from JSON. Equivalent in
// behavior to errors.New result; defined as a distinct type so
// callers using errors.As can detect a failure-rehydrated error if
// they need to.
type jsonError string

func (e jsonError) Error() string { return string(e) }
