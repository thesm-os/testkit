// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package visualize_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/trace"
	"go.thesmos.sh/testkit/core/visualize"
)

func TestEmitHeader(t *testing.T) {
	t.Parallel()

	t.Run("renders subject, generator, hex seed, and event count", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{StartNs: 0, EndNs: 100, Method: "Get", Component: "Ledger"})
		tr.Record(trace.Event{StartNs: 200, EndNs: 350, Method: "Put", Component: "Ledger"})

		tl := visualize.Timeline{
			Subject:   "ledger.Subsystem",
			Generator: "sim",
			Seed:      0xDEADBEEF,
			Trace:     tr,
		}

		var buf bytes.Buffer
		testkit.NoError(t, visualize.Emit(&buf, tl), "emit")

		testkit.Assert(t, buf.String()).
			Contains("ledger.Subsystem", "subject in header").
			Contains("sim", "generator in header").
			Contains("0x00000000DEADBEEF", "seed rendered as 16-hex").
			Contains("<strong>Events:</strong> 2", "event count in header")
	})
}

func TestEmitLanes(t *testing.T) {
	t.Parallel()

	t.Run("component-tagged events lane by Component, alphabetical", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{StartNs: 0, EndNs: 10, Method: "A", Component: "Scheduler"})
		tr.Record(trace.Event{StartNs: 20, EndNs: 30, Method: "B", Component: "Ledger"})

		tl := visualize.Timeline{
			Subject: "x", Generator: "model", Trace: tr,
			Style: visualize.Style{ComponentColors: map[string]string{"Ledger": "#abcdef"}},
		}

		var buf bytes.Buffer
		testkit.NoError(t, visualize.Emit(&buf, tl), "emit")
		out := buf.String()

		idxLedger := strings.Index(out, ">Ledger<")
		idxScheduler := strings.Index(out, ">Scheduler<")
		testkit.True(t, idxLedger > 0 && idxScheduler > 0, "both lane labels present")
		testkit.True(t, idxLedger < idxScheduler, "lanes sorted alphabetically (Ledger < Scheduler)")
		testkit.Assert(t, out).Contains("#abcdef", "override color applied to Ledger lane")
	})

	t.Run("untagged events fall back to goroutine then default", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{StartNs: 0, EndNs: 10, Method: "Run", Goroutine: 7})
		tr.Record(trace.Event{StartNs: 0, EndNs: 10, Method: "Run"})

		tl := visualize.Timeline{Subject: "x", Generator: "model", Trace: tr}

		var buf bytes.Buffer
		testkit.NoError(t, visualize.Emit(&buf, tl), "emit")

		testkit.Assert(t, buf.String()).
			Contains(">default<", "events without component or goroutine lane in default").
			Contains(">goroutine-7<", "events with goroutine but no component lane by GID")
	})
}

func TestEmitEvents(t *testing.T) {
	t.Parallel()

	t.Run("err class, tooltip, and table cells populate from event fields", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{StartNs: 0, EndNs: 5_000_000, Method: "Slow", Component: "L"})
		tr.Record(trace.Event{StartNs: 5_000_000, EndNs: 5_000_001, Method: "Fast", Component: "L"})
		tr.Record(trace.Event{
			StartNs: 6_000_000, EndNs: 6_500_000,
			Method: "Boom", Component: "L",
			Err:     errors.New("transient"),
			REQTags: []string{"REQ-1", "REQ-2"},
		})

		tl := visualize.Timeline{Subject: "x", Generator: "model", Trace: tr}

		var buf bytes.Buffer
		testkit.NoError(t, visualize.Emit(&buf, tl), "emit")

		testkit.Assert(t, buf.String()).
			Contains(`class="event err"`, "errored event gets err CSS class").
			Contains("L.Boom  err=transient  REQ=REQ-1,REQ-2", "tooltip combines context").
			Contains("REQ-1,REQ-2", "REQ tags rendered").
			Contains(`class="cell-err"`, "error cell class present")
	})

	t.Run("tooltip without component renders bare method", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{StartNs: 0, EndNs: 1, Method: "Foo"})
		var buf bytes.Buffer
		testkit.NoError(t, visualize.Emit(&buf, visualize.Timeline{Trace: tr}), "emit")
		testkit.Assert(t, buf.String()).Contains("<title>Foo</title>", "bare method tooltip")
	})

	t.Run("tooltip with component renders Component.Method", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{StartNs: 0, EndNs: 1, Method: "Foo", Component: "Bar"})
		var buf bytes.Buffer
		testkit.NoError(t, visualize.Emit(&buf, visualize.Timeline{Trace: tr}), "emit")
		testkit.Assert(t, buf.String()).Contains("<title>Bar.Foo</title>", "component-qualified tooltip")
	})

	t.Run("event width clamps to minWidth for sub-pixel durations", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{StartNs: 0, EndNs: 1_000_000_000, Method: "Wide", Component: "L"})
		tr.Record(trace.Event{StartNs: 500_000_000, EndNs: 500_000_010, Method: "Tiny", Component: "L"})

		var buf bytes.Buffer
		testkit.NoError(t, visualize.Emit(&buf, visualize.Timeline{Trace: tr}), "emit")
		testkit.Assert(t, buf.String()).Contains(`width="6"`, "tiny event clamped to minWidth")
	})
}

func TestEmitEmptyTrace(t *testing.T) {
	t.Parallel()

	t.Run("nil Trace renders zero-event canvas", func(t *testing.T) {
		t.Parallel()
		tl := visualize.Timeline{Subject: "x", Generator: "model"}
		var buf bytes.Buffer
		testkit.NoError(t, visualize.Emit(&buf, tl), "emit")
		testkit.Assert(t, buf.String()).Contains("<strong>Events:</strong> 0", "zero events")
	})

	t.Run("empty Trace uses fixed canvas width", func(t *testing.T) {
		t.Parallel()
		tl := visualize.Timeline{Subject: "x", Generator: "model", Trace: trace.New()}
		var buf bytes.Buffer
		testkit.NoError(t, visualize.Emit(&buf, tl), "emit")
		testkit.Assert(t, buf.String()).Contains(`width="980"`, "fixed canvas width 800+laneInset")
	})

	t.Run("out-of-order StartNs widens the timespan window", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{StartNs: 100, EndNs: 200, Method: "Late", Component: "L"})
		tr.Record(trace.Event{StartNs: 0, EndNs: 50, Method: "Early", Component: "L"})

		var buf bytes.Buffer
		testkit.NoError(t, visualize.Emit(&buf, visualize.Timeline{Trace: tr}), "emit")
		out := buf.String()

		idxEarly := strings.Index(out, ">L.Early</title>")
		idxLate := strings.Index(out, ">L.Late</title>")
		testkit.True(t, idxEarly > 0 && idxLate > 0, "both event tooltips present")
	})
}

func TestEmitDeterminism(t *testing.T) {
	t.Parallel()

	t.Run("byte-identical output for byte-identical inputs", func(t *testing.T) {
		t.Parallel()
		build := func() visualize.Timeline {
			tr := trace.New()
			tr.Record(trace.Event{StartNs: 0, EndNs: 100, Method: "A", Component: "Z"})
			tr.Record(trace.Event{StartNs: 100, EndNs: 200, Method: "B", Component: "A"})
			return visualize.Timeline{
				Subject:   "x",
				Generator: "model",
				Seed:      1,
				Trace:     tr,
				Overlays: []visualize.Overlay{
					visualize.REQOverlay(),
					visualize.CausalityOverlay(),
				},
			}
		}

		var a, b bytes.Buffer
		testkit.NoError(t, visualize.Emit(&a, build()), "emit a")
		testkit.NoError(t, visualize.Emit(&b, build()), "emit b")
		testkit.Equal(t, a.String(), b.String(), "byte-identical for byte-identical inputs")
	})
}
