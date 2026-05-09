// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package trace

// EventID uniquely identifies an [Event] within a [Trace]. IDs are
// assigned by [Trace.Record], monotonic from 1, never reused. Zero
// is the sentinel for "no event" — an [Event.Causality] entry of
// zero is invalid and surfaces in [Trace.ValidateCausality].
type EventID uint64

// Event is one method-call observation. The shape is shared across
// every generator runtime: model emits per-interface fragments;
// sim populates [Event.Component] and cross-component [Event.Causality];
// chaos populates [Event.FaultContext]; diff-rollout uses
// [Event.ClientID] for N-lane partitioning; replay consumes Events
// from any of these producers.
//
// Events are immutable once recorded. Filters and snapshots return
// independent copies, so consumers can range over filtered Traces
// without coordinating with the source's appender.
type Event struct {
	// ID is assigned by [Trace.Record]. Caller-supplied values are
	// overwritten. Monotonic from 1; zero is the no-event sentinel.
	ID EventID

	// Tick is the engine tick at which the event fired. Zero for
	// traces produced outside an engine context (per-interface
	// model traces use only the (StartNs, EndNs) timing).
	Tick int

	// StartNs and EndNs are engine-clock-relative for engine traces
	// and wall-clock-relative for non-engine traces. Both are
	// nanoseconds. EndNs >= StartNs is invariant.
	StartNs int64
	EndNs   int64

	// Goroutine is the engine-assigned worker ID, NOT the OS
	// goroutine ID. Zero means "single-worker" or "no worker
	// partition." Engine-assigned values start at 1.
	Goroutine int

	// ClientID partitions multi-client traces. Zero is the default
	// partition; concurrent runs assign 1..N to per-client lanes.
	// The diff-rollout runner uses ClientID for N-impl lockstep
	// partitioning.
	ClientID int

	// Component identifies the subsystem component that produced
	// the call. Empty for per-interface traces. The sim engine
	// populates Component for every wrapped-stub call.
	Component string

	// Method is the interface method name (e.g., "Get", "Put").
	Method string

	// Inputs is the slice of arguments passed to Method. Type-erased;
	// consumers re-type per-shape (linearize bridge, law evaluators).
	Inputs []any

	// Output is the method's return value. For methods returning
	// multiple values, consumers wrap them in a struct or tuple.
	Output any

	// Err is the method's error return, if any. Stored separately
	// from Output so callers don't have to box (V, error) into an
	// interface for the common case.
	Err error

	// Causality lists the IDs of events that happen-before this
	// one. Sim populates this for cross-interface causality (e.g.,
	// Ledger.Commit firing Scheduler.Schedule). Empty for events
	// with no recorded predecessors.
	Causality []EventID

	// REQTags carries requirement IDs traced through this event.
	// The model generator threads //testkit:req REQ-... directive
	// values into every emitted law and trace event.
	REQTags []string

	// FaultContext is non-nil when a fault was active at the time
	// of the call. Carries the active fault set and whether the
	// call was affected. Populated by the chaos engine.
	FaultContext *FaultContext

	// Metadata is generator-specific overlay data. The model
	// generator stores witness-extraction state; diff-rollout
	// stores per-lane comparison results; replay stores
	// trace-source provenance. Consumers ignore unknown keys.
	Metadata map[string]any
}

// FaultContext describes the fault state at the moment an [Event]
// was recorded. Populated by the chaos runtime when one or more
// faults are active; nil when no faults are scheduled or the call
// fired during a fault-free window.
type FaultContext struct {
	// Active lists every fault active during the call. A call may
	// be touched by multiple faults simultaneously (e.g., a
	// network partition AND a clock skew); each is listed.
	Active []FaultActivation

	// Affected reports whether any active fault actually mutated
	// this call's behavior. A fault may be active without
	// affecting a particular method (a NetworkPartition active on
	// component A doesn't affect calls to component B). The chaos
	// runtime sets Affected based on per-fault routing rules.
	Affected bool
}

// FaultActivation records a single active fault at event time. The
// chaos runtime emits one FaultActivation per active fault into the
// containing [FaultContext.Active].
type FaultActivation struct {
	// Name is the fault-primitive identifier (e.g.,
	// "NetworkPartition", "ClockSkew"). Matches the registered
	// name in testkit/registry/.Faults.
	Name string

	// Component is the subsystem component the fault targets, when
	// applicable. Empty for global faults (e.g., engine-wide clock
	// jumps).
	Component string

	// StartedAt is the engine tick at which the fault activated.
	StartedAt int

	// EndsAt is the engine tick at which the fault is scheduled to
	// heal. Negative one means "unbounded" (heals only on explicit
	// removal).
	EndsAt int

	// Args is the fault-primitive's per-activation parameters
	// (e.g., {"probability": 0.05} for MessageDrop). Stable across
	// activations of the same fault scheduled with the same args.
	Args map[string]any
}
