// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package trace

import (
	"slices"
	"sync"
	"time"
)

// Trace is a thread-safe append-only log of [Event] values. The
// zero value is not usable; construct via [New]. Filters return
// independent Traces over a snapshot of the source, so a filter
// taken before a Record call does not see events appended after.
type Trace struct {
	mu     sync.Mutex
	events []Event
	nextID EventID
}

// New constructs an empty [Trace].
func New() *Trace {
	return &Trace{nextID: 1}
}

// Record appends an event and returns its assigned ID. The
// caller-supplied [Event.ID] is overwritten; monotonic IDs from 1
// are stable across the Trace's lifetime. Thread-safe.
func (t *Trace) Record(e Event) EventID {
	t.mu.Lock()
	defer t.mu.Unlock()
	e.ID = t.nextID
	t.nextID++
	t.events = append(t.events, e)
	return e.ID
}

// RecordOp is a convenience that timestamps an operation and
// records it as a single event with empty causality, no fault
// context, and zero ClientID. Components driving sequential
// per-interface traces (model runner) use this; sim and chaos use
// the full [Trace.Record] path with explicit causality.
func (t *Trace) RecordOp(start time.Time, method string, inputs []any, output any, err error) EventID {
	end := time.Now()
	return t.Record(Event{
		StartNs: start.UnixNano(),
		EndNs:   end.UnixNano(),
		Method:  method,
		Inputs:  inputs,
		Output:  output,
		Err:     err,
	})
}

// Snapshot returns a copy of every recorded event in record order.
// The returned slice is independent of the Trace; subsequent Record
// calls do not affect it.
func (t *Trace) Snapshot() []Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]Event, len(t.events))
	copy(out, t.events)
	return out
}

// Len returns the number of recorded events.
func (t *Trace) Len() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.events)
}

// Reset clears every recorded event. The next [Trace.Record] call
// reassigns IDs from 1; consumers holding pre-Reset IDs see them
// become dangling references. The model runner calls Reset between
// rapid iterations so per-iteration IDs don't grow unbounded.
func (t *Trace) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = nil
	t.nextID = 1
}

// FilterByPredicate returns a new [Trace] containing every event
// from the source where pred returns true. Events are copied; the
// new Trace assigns fresh IDs starting at 1. Original IDs are
// preserved on each event's [Event.ID] field — the new Trace's
// nextID is the maximum source ID + 1 so further Record calls on
// the filtered Trace produce non-overlapping IDs.
//
// Every other Filter* method is a thin wrapper over this one.
func (t *Trace) FilterByPredicate(pred func(Event) bool) *Trace {
	src := t.Snapshot()
	out := New()
	out.mu.Lock()
	defer out.mu.Unlock()
	var maxID EventID
	for _, e := range src {
		if !pred(e) {
			continue
		}
		out.events = append(out.events, e)
		if e.ID > maxID {
			maxID = e.ID
		}
	}
	if maxID > 0 {
		out.nextID = maxID + 1
	}
	return out
}

// FilterByClient returns events whose [Event.ClientID] equals id.
// Used for per-client invariant routing in multi-client harnesses.
func (t *Trace) FilterByClient(id int) *Trace {
	return t.FilterByPredicate(func(e Event) bool { return e.ClientID == id })
}

// FilterByComponent returns events whose [Event.Component] equals
// name. Empty name selects events from per-interface traces.
func (t *Trace) FilterByComponent(name string) *Trace {
	return t.FilterByPredicate(func(e Event) bool { return e.Component == name })
}

// FilterByGoroutine returns events whose [Event.Goroutine] equals
// gid. The engine-assigned worker ID is matched, NOT the OS
// goroutine ID — testkit/sim's worker pool maps OS goroutines to
// engine IDs at registration.
func (t *Trace) FilterByGoroutine(gid int) *Trace {
	return t.FilterByPredicate(func(e Event) bool { return e.Goroutine == gid })
}

// FilterByMethod returns events whose [Event.Method] equals name.
func (t *Trace) FilterByMethod(name string) *Trace {
	return t.FilterByPredicate(func(e Event) bool { return e.Method == name })
}

// FilterByREQ returns events that carry the named requirement ID
// in [Event.REQTags]. The REQ-to-law coverage matrix uses this to
// build per-requirement event lists.
func (t *Trace) FilterByREQ(reqID string) *Trace {
	return t.FilterByPredicate(func(e Event) bool {
		return slices.Contains(e.REQTags, reqID)
	})
}

// FaultEvents returns events whose [Event.FaultContext] is non-nil.
// Used by chaos's load-bearing-fault extraction: walk the failure's
// causal hull, intersect with FaultEvents, ablate each fault in
// turn.
func (t *Trace) FaultEvents() *Trace {
	return t.FilterByPredicate(func(e Event) bool { return e.FaultContext != nil })
}

// CausalSlice returns the transitive causal hull of the named
// event: the event itself plus every event in its [Event.Causality]
// chain (and theirs, recursively). Order follows source recording
// order. Returns an empty Trace when id is not present in the
// source.
//
// The model generator uses CausalSlice for causal/dependency-aware
// shrinking: drop actions outside the hull of the failing
// assertion to produce a minimal reproducer.
func (t *Trace) CausalSlice(id EventID) *Trace {
	src := t.Snapshot()
	byID := make(map[EventID]Event, len(src))
	for _, e := range src {
		byID[e.ID] = e
	}
	if _, ok := byID[id]; !ok {
		return New()
	}
	keep := make(map[EventID]struct{})
	var visit func(EventID)
	visit = func(eid EventID) {
		if _, seen := keep[eid]; seen {
			return
		}
		e, ok := byID[eid]
		if !ok {
			return
		}
		keep[eid] = struct{}{}
		for _, pred := range e.Causality {
			visit(pred)
		}
	}
	visit(id)

	out := New()
	out.mu.Lock()
	defer out.mu.Unlock()
	var maxID EventID
	for _, e := range src {
		if _, ok := keep[e.ID]; !ok {
			continue
		}
		out.events = append(out.events, e)
		if e.ID > maxID {
			maxID = e.ID
		}
	}
	if maxID > 0 {
		out.nextID = maxID + 1
	}
	return out
}

// ValidateCausality reports any [Event.Causality] entries that
// reference IDs not present in the trace. Returns a slice of
// dangling-reference complaints, one per offending event. Empty
// slice means every causality edge resolves.
func (t *Trace) ValidateCausality() []DanglingRef {
	src := t.Snapshot()
	known := make(map[EventID]struct{}, len(src))
	for _, e := range src {
		known[e.ID] = struct{}{}
	}
	var out []DanglingRef
	for _, e := range src {
		for _, pred := range e.Causality {
			if pred == 0 {
				out = append(out, DanglingRef{Event: e.ID, Predecessor: pred, Reason: "zero is the no-event sentinel"})
				continue
			}
			if _, ok := known[pred]; !ok {
				out = append(out, DanglingRef{
					Event:       e.ID,
					Predecessor: pred,
					Reason:      "predecessor not present in trace",
				})
			}
		}
	}
	return out
}

// DanglingRef describes a causality edge that does not resolve to
// an event in the trace. Returned by [Trace.ValidateCausality]. Used
// by builtin invariant `TraceCausalityClosed` to fail runs whose
// engine recorded incomplete causality.
type DanglingRef struct {
	Event       EventID
	Predecessor EventID
	Reason      string
}
