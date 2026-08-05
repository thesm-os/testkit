// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package trace_test

import (
	"encoding/json"
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/trace"
)

func TestTraceJSONRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("preserves recorded events", func(t *testing.T) {
		t.Parallel()
		original := trace.New()
		original.Record(trace.Event{
			Method:    "Read",
			Tick:      1,
			ClientID:  2,
			Component: "Ledger",
			REQTags:   []string{"REQ-1"},
		})
		original.Record(trace.Event{
			Method: "Write",
			Tick:   2,
		})

		b, err := json.Marshal(original)
		testkit.NoError(t, err, "marshal")

		got := trace.New()
		err = json.Unmarshal(b, got)
		testkit.NoError(t, err, "unmarshal")

		testkit.Equal(t, got.Len(), 2, "two events recovered")
		events := got.Snapshot()
		testkit.Equal(t, events[0].Method, "Read", "first method")
		testkit.Equal(t, events[0].Tick, 1, "first tick")
		testkit.Equal(t, events[0].Component, "Ledger", "first component")
		testkit.Equal(t, events[1].Method, "Write", "second method")
	})

	t.Run("preserves causality references", func(t *testing.T) {
		t.Parallel()
		original := trace.New()
		id1 := original.Record(trace.Event{Method: "A"})
		original.Record(trace.Event{Method: "B", Causality: []trace.EventID{id1}})

		b, err := json.Marshal(original)
		testkit.NoError(t, err, "marshal")

		got := trace.New()
		err = json.Unmarshal(b, got)
		testkit.NoError(t, err, "unmarshal")

		events := got.Snapshot()
		testkit.Equal(t, len(events[1].Causality), 1, "causality preserved")
		testkit.Equal(t, events[1].Causality[0], id1, "causality predecessor")
	})

	t.Run("preserves error message", func(t *testing.T) {
		t.Parallel()
		original := trace.New()
		original.Record(trace.Event{Method: "X", Err: errors.New("kaboom")})

		b, err := json.Marshal(original)
		testkit.NoError(t, err, "marshal")

		got := trace.New()
		err = json.Unmarshal(b, got)
		testkit.NoError(t, err, "unmarshal")

		testkit.Equal(t, got.Snapshot()[0].Err.Error(), "kaboom", "error message")
	})

	t.Run("rebuilds nextID for further appends", func(t *testing.T) {
		t.Parallel()
		original := trace.New()
		original.Record(trace.Event{Method: "A"})
		original.Record(trace.Event{Method: "B"})

		b, err := json.Marshal(original)
		testkit.NoError(t, err, "marshal")

		got := trace.New()
		err = json.Unmarshal(b, got)
		testkit.NoError(t, err, "unmarshal")

		newID := got.Record(trace.Event{Method: "C"})
		testkit.Equal(t, newID, trace.EventID(3), "new ID continues from rebuilt counter")
	})

	t.Run("preserves fault context", func(t *testing.T) {
		t.Parallel()
		original := trace.New()
		original.Record(trace.Event{
			Method: "X",
			FaultContext: &trace.FaultContext{
				Affected: true,
				Active: []trace.FaultActivation{
					{Name: "NetworkPartition", Component: "Ledger", StartedAt: 5, EndsAt: 10},
				},
			},
		})

		b, err := json.Marshal(original)
		testkit.NoError(t, err, "marshal")

		got := trace.New()
		err = json.Unmarshal(b, got)
		testkit.NoError(t, err, "unmarshal")

		fc := got.Snapshot()[0].FaultContext
		testkit.True(t, fc != nil, "FaultContext present")
		testkit.True(t, fc.Affected, "Affected flag")
		testkit.Equal(t, len(fc.Active), 1, "one active fault")
		testkit.Equal(t, fc.Active[0].Name, "NetworkPartition", "fault name")
	})

	t.Run("rejects mistyped events field on Trace", func(t *testing.T) {
		t.Parallel()
		// Valid outer JSON but events isn't an array — exercises
		// the inner json.Unmarshal error path inside the custom
		// UnmarshalJSON method (rather than stdlib's tokenizer).
		got := trace.New()
		err := json.Unmarshal([]byte(`{"events":"not-an-array"}`), got)
		testkit.True(t, err != nil, "must error")
	})

	t.Run("rejects mistyped Event field", func(t *testing.T) {
		t.Parallel()
		// Same shape: valid JSON, wrong inner type.
		var e trace.Event
		err := json.Unmarshal([]byte(`{"id":"not-a-number"}`), &e)
		testkit.True(t, err != nil, "must error")
	})

	t.Run("propagates marshal errors from unmarshalable values", func(t *testing.T) {
		t.Parallel()
		// Channels can't be marshaled to JSON. The error path on
		// the outer MarshalJSON wraps it with the trace package
		// prefix.
		tr := trace.New()
		tr.Record(trace.Event{
			Method:   "X",
			Metadata: map[string]any{"ch": make(chan int)},
		})
		_, err := json.Marshal(tr)
		testkit.True(t, err != nil, "must error on unmarshalable value")
	})

	t.Run("propagates Event marshal errors", func(t *testing.T) {
		t.Parallel()
		e := trace.Event{
			Method:   "X",
			Metadata: map[string]any{"ch": make(chan int)},
		}
		_, err := json.Marshal(e)
		testkit.True(t, err != nil, "must error on unmarshalable value")
	})
}
