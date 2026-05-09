// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package trace_test

import (
	"sync"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/model/trace"
)

func TestTrace(t *testing.T) {
	t.Parallel()

	t.Run("New creates empty trace", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		testkit.Equal(t, tr.Len(), 0, "empty trace has length 0")
		testkit.Len(t, tr.Snapshot(), 0, "empty trace snapshot is empty")
	})

	t.Run("Record appends events", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{OpName: "Get", ClientID: 0})
		tr.Record(trace.Event{OpName: "Put", ClientID: 1})
		testkit.Equal(t, tr.Len(), 2, "two events recorded")
	})

	t.Run("RecordOp timestamps and records", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		start := time.Now()
		tr.RecordOp(start, "Get", 0, []any{"key"}, "val", nil)
		testkit.Equal(t, tr.Len(), 1, "one event recorded")

		snap := tr.Snapshot()
		testkit.Equal(t, snap[0].OpName, "Get", "op name")
		testkit.Equal(t, snap[0].ClientID, 0, "client ID")
		testkit.True(t, snap[0].StartNs > 0, "start timestamp set")
		testkit.True(t, snap[0].EndNs >= snap[0].StartNs, "end >= start")
		testkit.Equal(t, snap[0].Output, any("val"), "output captured")
		testkit.True(t, snap[0].Err == nil, "no error")
	})

	t.Run("Snapshot returns defensive copy", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{OpName: "Get"})
		snap := tr.Snapshot()
		snap[0].OpName = "MUTATED"
		testkit.Equal(t, tr.Snapshot()[0].OpName, "Get",
			"mutation of snapshot must not affect trace")
	})

	t.Run("Reset clears all events", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{OpName: "Get"})
		tr.Record(trace.Event{OpName: "Put"})
		tr.Reset()
		testkit.Equal(t, tr.Len(), 0, "reset clears events")
		testkit.Len(t, tr.Snapshot(), 0, "reset snapshot is empty")
	})

	t.Run("concurrent Record is safe", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		const n = 100
		var wg sync.WaitGroup
		wg.Add(n)
		for i := range n {
			go func(id int) {
				defer wg.Done()
				tr.Record(trace.Event{OpName: "op", ClientID: id})
			}(i)
		}
		wg.Wait()
		testkit.Equal(t, tr.Len(), n, "all concurrent records landed")
	})
}
