// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package trace

import (
	"encoding/json"
	"fmt"
)

// MarshalJSON encodes the Trace as `{"events": [...]}`. The
// internal mutex and nextID counter are not serialized — a
// round-tripped Trace's [Trace.nextID] is rebuilt from the maximum
// recorded ID + 1, matching what fresh Record calls would produce.
func (t *Trace) MarshalJSON() ([]byte, error) {
	out := struct {
		Events []Event `json:"events"`
	}{Events: t.Snapshot()}
	b, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("trace.Trace.MarshalJSON: %w", err)
	}
	return b, nil
}

// UnmarshalJSON decodes the events slice and rebuilds the Trace's
// nextID. The result is safe for concurrent reads but should be
// treated as immutable post-decode unless the caller is willing to
// inherit the producer's nextID accounting.
func (t *Trace) UnmarshalJSON(b []byte) error {
	in := struct {
		Events []Event `json:"events"`
	}{}
	if err := json.Unmarshal(b, &in); err != nil {
		return fmt.Errorf("trace.Trace.UnmarshalJSON: %w", err)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = in.Events
	t.nextID = 1
	for _, e := range in.Events {
		if e.ID >= t.nextID {
			t.nextID = e.ID + 1
		}
	}
	return nil
}

// MarshalJSON for Event renders [Event.Err] as a string under the
// "err" key (omitted when nil). All other fields use the struct
// tags below.
func (e Event) MarshalJSON() ([]byte, error) {
	type alias Event
	out := struct {
		alias
		ErrMsg string `json:"err,omitempty"`
	}{alias: alias(e)}
	if e.Err != nil {
		out.ErrMsg = e.Err.Error()
	}
	// We never marshal the alias's Err field directly — it's the
	// untagged interface that produces the empty struct otherwise.
	out.Err = nil
	b, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("trace.Event.MarshalJSON: %w", err)
	}
	return b, nil
}

// UnmarshalJSON rehydrates an Event, restoring [Event.Err] from
// the "err" string field as a synthesized error value.
func (e *Event) UnmarshalJSON(b []byte) error {
	type alias Event
	in := struct {
		*alias
		ErrMsg string `json:"err,omitempty"`
	}{alias: (*alias)(e)}
	if err := json.Unmarshal(b, &in); err != nil {
		return fmt.Errorf("trace.Event.UnmarshalJSON: %w", err)
	}
	if in.ErrMsg != "" {
		e.Err = jsonError(in.ErrMsg)
	}
	return nil
}

// jsonError is a minimal error type used to carry a message
// recovered from JSON. Distinct from errors.New result so callers
// using errors.As can detect a JSON-rehydrated error if they need to.
type jsonError string

func (e jsonError) Error() string { return string(e) }
