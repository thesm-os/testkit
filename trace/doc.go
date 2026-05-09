// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package trace records method-call observations across testkit's
// generator runtimes. One [Event] per call carries enough context
// for cross-cutting analysis: failure classification, causal
// shrinking, replay-of-replay verification, fault correlation,
// per-client invariant routing.
//
// The model generator records per-interface trace fragments; the
// sim engine records subsystem-wide traces with [Event.Component]
// populated; chaos populates [Event.FaultContext] when a fault
// touches a recorded call; differential-rollout records N parallel
// lanes via the [Event.ClientID] partition; replay consumes Traces
// from any of these producers. The Trace package is the protocol
// between layers — every layer reads and writes the same shape.
//
// [Trace] is thread-safe append-only. Filter operations
// ([Trace.FilterByClient], [Trace.FilterByComponent], …) return new
// independent Traces over a snapshot of the source events, so
// concurrent appends to the source after a filter call do not
// affect the filtered view.
//
// [EqualForDeterminism] compares two Traces under the normalization
// rules the determinism cross-validation gate uses: per-event field
// equality with wall-clock-only fields stripped. Generator-emitted
// regression tests use it to verify same-seed → same-trace.
package trace
