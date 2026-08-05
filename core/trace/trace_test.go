// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package trace_test

import (
	"errors"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/trace"
)

func TestRecord(t *testing.T) {
	t.Parallel()

	t.Run("assigns monotonic IDs from 1", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		id1 := tr.Record(trace.Event{Method: "A"})
		id2 := tr.Record(trace.Event{Method: "B"})
		id3 := tr.Record(trace.Event{Method: "C"})
		testkit.Equal(t, id1, trace.EventID(1), "first ID is 1")
		testkit.Equal(t, id2, trace.EventID(2), "monotonic")
		testkit.Equal(t, id3, trace.EventID(3), "monotonic")
	})

	t.Run("overwrites caller-supplied ID", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		id := tr.Record(trace.Event{ID: 999, Method: "X"})
		testkit.Equal(t, id, trace.EventID(1), "Record assigns its own ID")
		got := tr.Snapshot()
		testkit.Equal(t, len(got), 1, "one event recorded")
		testkit.Equal(t, got[0].ID, trace.EventID(1), "stored ID matches assigned ID")
	})

	t.Run("appends in record order", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{Method: "first"})
		tr.Record(trace.Event{Method: "second"})
		tr.Record(trace.Event{Method: "third"})
		got := tr.Snapshot()
		testkit.Equal(t, len(got), 3, "three events recorded")
		testkit.Equal(t, got[0].Method, "first", "first")
		testkit.Equal(t, got[1].Method, "second", "second")
		testkit.Equal(t, got[2].Method, "third", "third")
	})

	t.Run("RecordOp timestamps and stores method/inputs/output", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		start := time.Now()
		id := tr.RecordOp(start, "Read", []any{"key"}, "value", nil)
		testkit.Equal(t, id, trace.EventID(1), "first ID")
		got := tr.Snapshot()
		testkit.Equal(t, got[0].Method, "Read", "method")
		testkit.Equal(t, got[0].Inputs, []any{"key"}, "inputs")
		testkit.Equal(t, got[0].Output, any("value"), "output")
		testkit.NoError(t, got[0].Err, "no error")
		testkit.True(t, got[0].EndNs >= got[0].StartNs, "EndNs >= StartNs")
	})
}

func TestSnapshotIsIndependent(t *testing.T) {
	t.Parallel()

	tr := trace.New()
	tr.Record(trace.Event{Method: "a"})
	tr.Record(trace.Event{Method: "b"})

	snap := tr.Snapshot()
	testkit.Equal(t, len(snap), 2, "two events captured")

	tr.Record(trace.Event{Method: "c"})
	testkit.Equal(t, len(snap), 2, "snapshot unaffected by later Record")
	testkit.Equal(t, tr.Len(), 3, "source has three events")
}

func TestLen(t *testing.T) {
	t.Parallel()

	tr := trace.New()
	testkit.Equal(t, tr.Len(), 0, "empty trace")
	tr.Record(trace.Event{Method: "a"})
	tr.Record(trace.Event{Method: "b"})
	testkit.Equal(t, tr.Len(), 2, "two events")
}

func TestReset(t *testing.T) {
	t.Parallel()

	t.Run("clears events", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{Method: "a"})
		tr.Record(trace.Event{Method: "b"})
		tr.Reset()
		testkit.Equal(t, tr.Len(), 0, "no events after Reset")
	})

	t.Run("resets ID counter to 1", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{Method: "a"})
		tr.Record(trace.Event{Method: "b"})
		tr.Reset()
		id := tr.Record(trace.Event{Method: "c"})
		testkit.Equal(t, id, trace.EventID(1), "next ID is 1 after Reset")
	})
}

func TestFilterByPredicate(t *testing.T) {
	t.Parallel()

	tr := trace.New()
	tr.Record(trace.Event{Method: "Read", ClientID: 1})
	tr.Record(trace.Event{Method: "Write", ClientID: 2})
	tr.Record(trace.Event{Method: "Read", ClientID: 3})

	t.Run("retains matching events in source order", func(t *testing.T) {
		t.Parallel()
		got := tr.FilterByPredicate(func(e trace.Event) bool { return e.Method == "Read" }).Snapshot()
		testkit.Equal(t, len(got), 2, "two Read events")
		testkit.Equal(t, got[0].Method, "Read", "first")
		testkit.Equal(t, got[1].Method, "Read", "second")
	})

	t.Run("preserves source IDs on filtered events", func(t *testing.T) {
		t.Parallel()
		got := tr.FilterByPredicate(func(e trace.Event) bool { return e.Method == "Write" }).Snapshot()
		testkit.Equal(t, len(got), 1, "one Write event")
		testkit.Equal(t, got[0].ID, trace.EventID(2), "preserves source ID")
	})

	t.Run("filtered Trace can be appended to without ID collision", func(t *testing.T) {
		t.Parallel()
		filtered := tr.FilterByPredicate(func(e trace.Event) bool { return e.Method == "Read" })
		newID := filtered.Record(trace.Event{Method: "appended"})
		testkit.True(t, newID > trace.EventID(3), "new ID exceeds max source ID")
	})

	t.Run("returns empty trace when nothing matches", func(t *testing.T) {
		t.Parallel()
		got := tr.FilterByPredicate(func(e trace.Event) bool { return false }).Snapshot()
		testkit.Equal(t, len(got), 0, "empty result")
	})
}

func TestFilterByClient(t *testing.T) {
	t.Parallel()

	tr := trace.New()
	tr.Record(trace.Event{Method: "A", ClientID: 0})
	tr.Record(trace.Event{Method: "B", ClientID: 1})
	tr.Record(trace.Event{Method: "C", ClientID: 1})
	tr.Record(trace.Event{Method: "D", ClientID: 2})

	got := tr.FilterByClient(1).Snapshot()
	testkit.Equal(t, len(got), 2, "two client-1 events")
	testkit.Equal(t, got[0].Method, "B", "first")
	testkit.Equal(t, got[1].Method, "C", "second")
}

func TestFilterByComponent(t *testing.T) {
	t.Parallel()

	tr := trace.New()
	tr.Record(trace.Event{Method: "Commit", Component: "Ledger"})
	tr.Record(trace.Event{Method: "Schedule", Component: "Scheduler"})
	tr.Record(trace.Event{Method: "Read", Component: "Ledger"})

	got := tr.FilterByComponent("Ledger").Snapshot()
	testkit.Equal(t, len(got), 2, "two Ledger events")
}

func TestFilterByGoroutine(t *testing.T) {
	t.Parallel()

	tr := trace.New()
	tr.Record(trace.Event{Method: "A", Goroutine: 1})
	tr.Record(trace.Event{Method: "B", Goroutine: 2})
	tr.Record(trace.Event{Method: "C", Goroutine: 1})

	got := tr.FilterByGoroutine(1).Snapshot()
	testkit.Equal(t, len(got), 2, "two goroutine-1 events")
}

func TestFilterByMethod(t *testing.T) {
	t.Parallel()

	tr := trace.New()
	tr.Record(trace.Event{Method: "Read"})
	tr.Record(trace.Event{Method: "Write"})
	tr.Record(trace.Event{Method: "Read"})

	got := tr.FilterByMethod("Read").Snapshot()
	testkit.Equal(t, len(got), 2, "two Read events")
}

func TestFilterByREQ(t *testing.T) {
	t.Parallel()

	tr := trace.New()
	tr.Record(trace.Event{Method: "A", REQTags: []string{"REQ-1", "REQ-2"}})
	tr.Record(trace.Event{Method: "B", REQTags: []string{"REQ-2"}})
	tr.Record(trace.Event{Method: "C", REQTags: []string{"REQ-3"}})

	t.Run("matches single tag", func(t *testing.T) {
		t.Parallel()
		got := tr.FilterByREQ("REQ-1").Snapshot()
		testkit.Equal(t, len(got), 1, "one event tagged REQ-1")
	})

	t.Run("matches event with multiple tags", func(t *testing.T) {
		t.Parallel()
		got := tr.FilterByREQ("REQ-2").Snapshot()
		testkit.Equal(t, len(got), 2, "two events tagged REQ-2")
	})

	t.Run("returns empty when tag absent", func(t *testing.T) {
		t.Parallel()
		got := tr.FilterByREQ("REQ-99").Snapshot()
		testkit.Equal(t, len(got), 0, "no events tagged REQ-99")
	})
}

func TestFaultEvents(t *testing.T) {
	t.Parallel()

	tr := trace.New()
	tr.Record(trace.Event{Method: "A"})
	tr.Record(trace.Event{Method: "B", FaultContext: &trace.FaultContext{Affected: true}})
	tr.Record(trace.Event{Method: "C"})

	got := tr.FaultEvents().Snapshot()
	testkit.Equal(t, len(got), 1, "one fault event")
	testkit.Equal(t, got[0].Method, "B", "the fault event")
}

func TestCausalSlice(t *testing.T) {
	t.Parallel()

	t.Run("returns the named event and its predecessors", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		id1 := tr.Record(trace.Event{Method: "A"})
		id2 := tr.Record(trace.Event{Method: "B", Causality: []trace.EventID{id1}})
		id3 := tr.Record(trace.Event{Method: "C", Causality: []trace.EventID{id2}})
		_ = tr.Record(trace.Event{Method: "D"}) // unrelated

		got := tr.CausalSlice(id3).Snapshot()
		testkit.Equal(t, len(got), 3, "A + B + C in causal hull")
		testkit.Equal(t, got[0].Method, "A", "predecessor first (source order)")
		testkit.Equal(t, got[1].Method, "B", "intermediate")
		testkit.Equal(t, got[2].Method, "C", "target last")
	})

	t.Run("handles diamond causality without duplicates", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		idA := tr.Record(trace.Event{Method: "A"})
		idB := tr.Record(trace.Event{Method: "B", Causality: []trace.EventID{idA}})
		idC := tr.Record(trace.Event{Method: "C", Causality: []trace.EventID{idA}})
		idD := tr.Record(trace.Event{Method: "D", Causality: []trace.EventID{idB, idC}})

		got := tr.CausalSlice(idD).Snapshot()
		testkit.Equal(t, len(got), 4, "A, B, C, D — A appears once even though both B and C reference it")
	})

	t.Run("returns empty Trace when ID not present", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{Method: "A"})
		got := tr.CausalSlice(trace.EventID(99)).Snapshot()
		testkit.Equal(t, len(got), 0, "unknown ID yields empty slice")
	})

	t.Run("tolerates dangling predecessor references", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		id := tr.Record(trace.Event{Method: "X", Causality: []trace.EventID{99}})
		got := tr.CausalSlice(id).Snapshot()
		testkit.Equal(t, len(got), 1, "only the named event survives a dangling reference")
	})
}

func TestValidateCausality(t *testing.T) {
	t.Parallel()

	t.Run("clean trace returns no complaints", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		id1 := tr.Record(trace.Event{Method: "A"})
		tr.Record(trace.Event{Method: "B", Causality: []trace.EventID{id1}})
		got := tr.ValidateCausality()
		testkit.Equal(t, len(got), 0, "no dangling refs")
	})

	t.Run("flags zero predecessor IDs", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{Method: "X", Causality: []trace.EventID{0}})
		got := tr.ValidateCausality()
		testkit.Equal(t, len(got), 1, "one dangling ref")
		testkit.Equal(t, got[0].Predecessor, trace.EventID(0), "zero predecessor")
	})

	t.Run("flags unknown predecessors", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{Method: "X", Causality: []trace.EventID{99}})
		got := tr.ValidateCausality()
		testkit.Equal(t, len(got), 1, "one dangling ref")
		testkit.Equal(t, got[0].Predecessor, trace.EventID(99), "unknown predecessor")
	})
}

func TestEqualForDeterminism(t *testing.T) {
	t.Parallel()

	t.Run("identical traces are determinism-equivalent", func(t *testing.T) {
		t.Parallel()
		a := trace.New()
		b := trace.New()
		for _, m := range []string{"X", "Y", "Z"} {
			a.Record(trace.Event{Method: m, Tick: 1})
			b.Record(trace.Event{Method: m, Tick: 1})
		}
		testkit.True(t, trace.EqualForDeterminism(a, b), "identical")
		testkit.Equal(t, trace.DiffForDeterminism(a, b), "", "no diff")
	})

	t.Run("differing methods are not equivalent", func(t *testing.T) {
		t.Parallel()
		a := trace.New()
		b := trace.New()
		a.Record(trace.Event{Method: "X"})
		b.Record(trace.Event{Method: "Y"})
		testkit.False(t, trace.EqualForDeterminism(a, b), "diverges")
		testkit.Assert(t, trace.DiffForDeterminism(a, b)).Contains("X", "diff cites left")
	})

	t.Run("equates nil and empty causality slices", func(t *testing.T) {
		t.Parallel()
		a := trace.New()
		b := trace.New()
		a.Record(trace.Event{Method: "X", Causality: nil})
		b.Record(trace.Event{Method: "X", Causality: []trace.EventID{}})
		testkit.True(t, trace.EqualForDeterminism(a, b),
			"nil and empty Causality compare equal")
	})

	t.Run("compares errors by message text", func(t *testing.T) {
		t.Parallel()
		a := trace.New()
		b := trace.New()
		a.Record(trace.Event{Method: "X", Err: errors.New("boom")})
		b.Record(trace.Event{Method: "X", Err: errors.New("boom")})
		testkit.True(t, trace.EqualForDeterminism(a, b),
			"distinct error values with same message are equivalent")
	})

	t.Run("differing error messages diverge", func(t *testing.T) {
		t.Parallel()
		a := trace.New()
		b := trace.New()
		a.Record(trace.Event{Method: "X", Err: errors.New("boom")})
		b.Record(trace.Event{Method: "X", Err: errors.New("ouch")})
		testkit.False(t, trace.EqualForDeterminism(a, b), "different messages diverge")
	})

	t.Run("nil vs non-nil error diverges", func(t *testing.T) {
		t.Parallel()
		a := trace.New()
		b := trace.New()
		a.Record(trace.Event{Method: "X"})
		b.Record(trace.Event{Method: "X", Err: errors.New("boom")})
		testkit.False(t, trace.EqualForDeterminism(a, b), "nil vs error diverges")
	})

	t.Run("non-nil vs nil error diverges (symmetry)", func(t *testing.T) {
		t.Parallel()
		a := trace.New()
		b := trace.New()
		a.Record(trace.Event{Method: "X", Err: errors.New("boom")})
		b.Record(trace.Event{Method: "X"})
		testkit.False(t, trace.EqualForDeterminism(a, b), "error vs nil diverges")
	})

	t.Run("differing tick diverges", func(t *testing.T) {
		t.Parallel()
		a := trace.New()
		b := trace.New()
		a.Record(trace.Event{Method: "X", Tick: 1})
		b.Record(trace.Event{Method: "X", Tick: 2})
		testkit.False(t, trace.EqualForDeterminism(a, b),
			"engine-relative tick is part of the determinism contract")
	})
}
