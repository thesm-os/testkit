// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package failure carries the unified failure envelope every
// generator runtime emits when an invariant fires, an action errors,
// a divergence is detected, or a chaos run crashes.
//
// One [Failure] envelope. Each generator sets [Failure.Generator]
// to its name ("model" / "sim" / "chaos" / "diff-rollout" / "replay")
// and populates [Failure.Details] with a structured payload specific
// to that layer. CI tooling — PR bots, REQ coverage matrices,
// certification queries — consumes one schema, regardless of which
// generator produced the failure.
//
// Trace + snapshot + artifacts attach to the envelope: [Failure.Trace]
// is the [*trace.Trace] captured at failure time, [Failure.Snapshot]
// holds per-component or per-impl state, and [Failure.Artifacts]
// references on-disk dumps (failfiles, Porcupine HTML, timeline
// renders, classified-failure JSON itself).
//
// Round-trip JSON serialization is supported on [Failure] —
// generator-emitted failure-<seed>.json artifacts re-parse into the
// same envelope, so CI ingestion is one decode step regardless of
// origin.
package failure
