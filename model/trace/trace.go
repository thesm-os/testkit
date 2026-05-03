// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package trace records (start_ns, end_ns, op, result) events for
// model-based test runs. Sequential mode optionally captures traces;
// concurrent mode (Pillar 3) feeds them to Porcupine for
// linearizability checking.
package trace

import (
	"sync"
	"time"
)

// Event records a single operation against the SUT.
type Event struct {
	StartNs  int64
	EndNs    int64
	OpName   string
	ClientID int   // 0 for sequential; goroutine index for concurrent
	Inputs   []any // type-erased; Porcupine bridge re-types
	Output   any
	Err      error
}

// Trace is a thread-safe append-only log of [Event] values.
type Trace struct {
	mu     sync.Mutex
	events []Event
}

// New creates an empty [Trace].
func New() *Trace {
	return &Trace{}
}

// Record appends an event. Thread-safe.
func (t *Trace) Record(e Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, e)
}

// RecordOp is a convenience that timestamps an operation.
// Call Start before the op, then RecordOp after with the result.
func (t *Trace) RecordOp(start time.Time, opName string, clientID int, inputs []any, output any, err error) {
	end := time.Now()
	t.Record(Event{
		StartNs:  start.UnixNano(),
		EndNs:    end.UnixNano(),
		OpName:   opName,
		ClientID: clientID,
		Inputs:   inputs,
		Output:   output,
		Err:      err,
	})
}

// Snapshot returns a copy of all recorded events.
func (t *Trace) Snapshot() []Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	cp := make([]Event, len(t.events))
	copy(cp, t.events)
	return cp
}

// Len returns the number of recorded events.
func (t *Trace) Len() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.events)
}

// Reset clears all recorded events.
func (t *Trace) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = nil
}
