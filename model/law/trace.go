// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law

import (
	"fmt"
	"slices"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/trace"
)

// TraceBinder is implemented by laws that need access to the
// per-iteration trace. The runner detects this interface and calls
// BindTrace at iteration start with the runner's trace buffer.
type TraceBinder interface {
	BindTrace(t *trace.Trace)
}

// AfterEvery checks that a predicate holds after every occurrence of
// the named action in the trace. The predicate receives the SUT after
// the action ran — it should verify the post-condition.
//
// Example: "after every Put, Count must equal the number of unique keys."
type AfterEvery[T any] struct {
	// ActionName is the action to observe.
	ActionName string

	// Predicate is called after every occurrence of ActionName.
	// Return non-nil to indicate a violation.
	Predicate func(rt *rapid.T, sut, ref T) error

	// Trace is the per-iteration trace buffer shared with the runner.
	Trace *trace.Trace
}

// BindTrace sets the trace reference. Called by the runner at
// iteration start.
func (l *AfterEvery[T]) BindTrace(t *trace.Trace) { l.Trace = t }

// ID returns a stable identifier.
func (l *AfterEvery[T]) ID() string {
	return "TRACE-AFTER-EVERY-" + l.ActionName
}

// REQID returns empty (consumer-supplied law).
func (*AfterEvery[T]) REQID() string { return "" }

// Check is called after every action. It inspects the last trace
// event to determine if the target action just ran.
func (l *AfterEvery[T]) Check(rt *rapid.T, sut, ref T) error {
	events := l.Trace.Snapshot()
	if len(events) == 0 {
		return nil
	}
	last := events[len(events)-1]
	if last.OpName != l.ActionName {
		return nil
	}
	err := l.Predicate(rt, sut, ref)
	if err != nil {
		return fmt.Errorf("AfterEvery(%s): %w", l.ActionName, err)
	}
	return nil
}

// EventuallyAfter checks that after the trigger action fires, the
// response action must appear within N steps. Useful for interfaces
// with causal ordering requirements (e.g., "after Write, Flush must
// fire within N ops").
//
// Note: since the model runner selects actions randomly, this
// combinator is meaningful only when the action set is small enough
// and the budget generous enough that the probability of the response
// not appearing is negligible. For large action sets, consider
// AfterEvery instead.
type EventuallyAfter[T any] struct {
	// Trigger is the action that starts the countdown.
	Trigger string

	// Response is the action that must appear within WithinSteps.
	Response string

	// WithinSteps is the maximum number of actions between trigger
	// and response. If exceeded, the law fires.
	WithinSteps int

	// Trace is the per-iteration trace buffer shared with the runner.
	Trace *trace.Trace
}

// BindTrace sets the trace reference.
func (l *EventuallyAfter[T]) BindTrace(t *trace.Trace) { l.Trace = t }

// ID returns a stable identifier.
func (l *EventuallyAfter[T]) ID() string {
	return fmt.Sprintf("TRACE-EVENTUALLY-%s-AFTER-%s", l.Response, l.Trigger)
}

// REQID returns empty (consumer-supplied law).
func (*EventuallyAfter[T]) REQID() string { return "" }

// Check is called after every action. It scans the trace for
// unsatisfied trigger→response pairs.
func (l *EventuallyAfter[T]) Check(_ *rapid.T, _, _ T) error {
	events := l.Trace.Snapshot()
	// Walk backwards from the end looking for the most recent trigger
	// that hasn't been followed by a response within WithinSteps.
	lastTriggerIdx := -1
	for i, v := range slices.Backward(events) {
		if v.OpName == l.Trigger {
			lastTriggerIdx = i
			break
		}
		if v.OpName == l.Response {
			// Response found before trigger — satisfied.
			return nil
		}
	}
	if lastTriggerIdx < 0 {
		return nil // no trigger fired yet
	}
	// Count steps since trigger.
	stepsSinceTrigger := len(events) - 1 - lastTriggerIdx
	if stepsSinceTrigger <= l.WithinSteps {
		return nil // still within budget
	}
	// Check if response appeared between trigger and now.
	for i := lastTriggerIdx + 1; i < len(events); i++ {
		if events[i].OpName == l.Response {
			return nil // satisfied
		}
	}
	return fmt.Errorf("EventuallyAfter: %s fired at step %d, %s not seen within %d steps (now at step %d)",
		l.Trigger, lastTriggerIdx, l.Response, l.WithinSteps, len(events)-1)
}

// Never checks that the named action never appears in the trace.
// Useful for asserting forbidden state transitions.
type Never[T any] struct {
	// ActionName is the action that must never occur.
	ActionName string

	// Trace is the per-iteration trace buffer shared with the runner.
	Trace *trace.Trace
}

// BindTrace sets the trace reference.
func (l *Never[T]) BindTrace(t *trace.Trace) { l.Trace = t }

// ID returns a stable identifier.
func (l *Never[T]) ID() string {
	return "TRACE-NEVER-" + l.ActionName
}

// REQID returns empty (consumer-supplied law).
func (*Never[T]) REQID() string { return "" }

// Check is called after every action. If the forbidden action appears
// in the trace, it fires.
func (l *Never[T]) Check(_ *rapid.T, _, _ T) error {
	events := l.Trace.Snapshot()
	for i, ev := range events {
		if ev.OpName == l.ActionName {
			return fmt.Errorf("never(%s): forbidden action appeared at step %d", l.ActionName, i)
		}
	}
	return nil
}
