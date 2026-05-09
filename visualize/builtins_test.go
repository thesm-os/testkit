// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package visualize_test

import (
	"bytes"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/failure"
	"go.thesmos.sh/testkit/trace"
	"go.thesmos.sh/testkit/visualize"
)

func TestFaultOverlay(t *testing.T) {
	t.Parallel()

	t.Run("Name returns fault", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, visualize.FaultOverlay().Name(), "fault", "overlay name")
	})

	t.Run("nil trace renders no markers", func(t *testing.T) {
		t.Parallel()
		o := visualize.FaultOverlay()
		testkit.Equal(t, len(o.Render(nil)), 0, "nil-trace render")
	})

	t.Run("events without FaultContext are skipped", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{StartNs: 0, EndNs: 1, Method: "A"})
		tr.Record(trace.Event{StartNs: 1, EndNs: 2, Method: "B", FaultContext: &trace.FaultContext{Affected: false}})
		testkit.Equal(t, len(visualize.FaultOverlay().Render(tr)), 0, "no affected events")
	})

	t.Run("affected events emit one marker each with derived style and tooltip", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{
			StartNs: 0, EndNs: 1, Method: "A",
			FaultContext: &trace.FaultContext{
				Affected: true,
				Active: []trace.FaultActivation{
					{Name: "Network Partition", Component: "Ledger"},
					{Name: "Clock Skew"},
				},
			},
		})
		markers := visualize.FaultOverlay().Render(tr)
		testkit.Equal(t, len(markers), 1, "one marker per affected event")
		testkit.Equal(t, markers[0].Layer, "fault", "layer")
		testkit.Equal(t, markers[0].Style, "network-partition", "style derived from first fault, lowercased and dashed")
		testkit.Assert(t, markers[0].Tooltip).
			Contains("Network Partition@Ledger", "first fault names component").
			Contains("Clock Skew", "second fault listed").
			Contains(",", "tooltip joins multiple")
	})

	t.Run("affected event with no Active list emits generic style", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{
			StartNs: 0, EndNs: 1, Method: "A",
			FaultContext: &trace.FaultContext{Affected: true},
		})
		markers := visualize.FaultOverlay().Render(tr)
		testkit.Equal(t, len(markers), 1, "one marker")
		testkit.Equal(t, markers[0].Style, "generic", "no-Active fallback style")
		testkit.Equal(t, markers[0].Tooltip, "fault active", "no-Active fallback tooltip")
	})
}

func TestDivergenceOverlay(t *testing.T) {
	t.Parallel()

	t.Run("Name returns divergence", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, visualize.DivergenceOverlay(nil).Name(), "divergence", "overlay name")
	})

	t.Run("renders one marker per DivergenceMarker with sorted lane names", func(t *testing.T) {
		t.Parallel()
		divs := []visualize.DivergenceMarker{
			{
				EventID: 5,
				Lanes:   map[string]any{"v2": 99, "v1": 42},
				Diff:    "v1=42 vs v2=99",
			},
		}
		markers := visualize.DivergenceOverlay(divs).Render(nil)
		testkit.Equal(t, len(markers), 1, "one marker")
		testkit.Equal(t, markers[0].Layer, "divergence", "layer")
		testkit.Equal(t, markers[0].EventID, trace.EventID(5), "event id")
		testkit.Assert(t, markers[0].Tooltip).
			Contains("v1 vs v2", "lane names sorted alphabetically").
			Contains("v1=42 vs v2=99", "diff appended")
	})

	t.Run("empty Diff omits the trailing dash", func(t *testing.T) {
		t.Parallel()
		divs := []visualize.DivergenceMarker{
			{EventID: 1, Lanes: map[string]any{"v1": 1, "v2": 1}},
		}
		markers := visualize.DivergenceOverlay(divs).Render(nil)
		testkit.Assert(t, markers[0].Tooltip).NotContains("—", "no diff separator when Diff empty")
	})
}

func TestCausalityOverlay(t *testing.T) {
	t.Parallel()

	t.Run("Name returns causality", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, visualize.CausalityOverlay().Name(), "causality", "overlay name")
	})

	t.Run("nil trace renders no markers", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, len(visualize.CausalityOverlay().Render(nil)), 0, "nil-trace render")
	})

	t.Run("events with no Causality are skipped", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{StartNs: 0, EndNs: 1, Method: "A"})
		testkit.Equal(t, len(visualize.CausalityOverlay().Render(tr)), 0, "no causal events")
	})

	t.Run("each event with predecessors gets a marker carrying the IDs", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		first := tr.Record(trace.Event{StartNs: 0, EndNs: 1, Method: "A"})
		tr.Record(trace.Event{StartNs: 2, EndNs: 3, Method: "B", Causality: []trace.EventID{first}})
		markers := visualize.CausalityOverlay().Render(tr)
		testkit.Equal(t, len(markers), 1, "one marker for the consequent")
		testkit.Equal(t, markers[0].Layer, "causality", "layer")
		preds, ok := markers[0].Metadata["predecessors"].([]trace.EventID)
		testkit.True(t, ok, "metadata carries typed predecessors slice")
		testkit.Equal(t, len(preds), 1, "one predecessor")
		testkit.Equal(t, preds[0], first, "predecessor matches recorded ID")
	})
}

func TestREQOverlay(t *testing.T) {
	t.Parallel()

	t.Run("Name returns req", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, visualize.REQOverlay().Name(), "req", "overlay name")
	})

	t.Run("nil trace renders no markers", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, len(visualize.REQOverlay().Render(nil)), 0, "nil-trace render")
	})

	t.Run("events without REQTags are skipped", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{StartNs: 0, EndNs: 1, Method: "A"})
		testkit.Equal(t, len(visualize.REQOverlay().Render(tr)), 0, "no tagged events")
	})

	t.Run("tagged events emit a marker per event", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{StartNs: 0, EndNs: 1, Method: "A", REQTags: []string{"REQ-1", "REQ-2"}})
		markers := visualize.REQOverlay().Render(tr)
		testkit.Equal(t, len(markers), 1, "one marker")
		testkit.Equal(t, markers[0].Style, "tag", "style")
		testkit.Assert(t, markers[0].Tooltip).Contains("REQ-1, REQ-2", "tags joined")
	})
}

func TestReplayMarkerOverlay(t *testing.T) {
	t.Parallel()

	t.Run("Name returns replay", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, visualize.ReplayMarkerOverlay().Name(), "replay", "overlay name")
	})

	t.Run("nil trace renders no markers", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, len(visualize.ReplayMarkerOverlay().Render(nil)), 0, "nil-trace render")
	})

	t.Run("events without replay_origin metadata are skipped", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{StartNs: 0, EndNs: 1, Method: "A"})
		tr.Record(trace.Event{StartNs: 1, EndNs: 2, Method: "B", Metadata: map[string]any{"other": 1}})
		testkit.Equal(t, len(visualize.ReplayMarkerOverlay().Render(tr)), 0, "no replay events")
	})

	t.Run("tagged replay events emit markers carrying the origin", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{
			StartNs: 0, EndNs: 1, Method: "A",
			Metadata: map[string]any{"replay_origin": "src-event-7"},
		})
		markers := visualize.ReplayMarkerOverlay().Render(tr)
		testkit.Equal(t, len(markers), 1, "one marker")
		testkit.Equal(t, markers[0].Style, "input", "style")
		testkit.Assert(t, markers[0].Tooltip).Contains("src-event-7", "origin in tooltip")
		testkit.Equal(t, markers[0].Metadata["origin"], "src-event-7", "origin in metadata")
	})
}

func TestSnapshotOverlay(t *testing.T) {
	t.Parallel()

	t.Run("Name returns snapshot", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, visualize.SnapshotOverlay(nil).Name(), "snapshot", "overlay name")
	})

	t.Run("nil trace renders no markers", func(t *testing.T) {
		t.Parallel()
		snap := &failure.Snapshot{PerComponent: map[string]any{"A": 1}}
		testkit.Equal(t, len(visualize.SnapshotOverlay(snap).Render(nil)), 0, "nil-trace render")
	})

	t.Run("nil snapshot renders no markers", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{StartNs: 0, EndNs: 1, Method: "A"})
		testkit.Equal(t, len(visualize.SnapshotOverlay(nil).Render(tr)), 0, "nil-snap render")
	})

	t.Run("empty snapshot renders no markers", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{StartNs: 0, EndNs: 1, Method: "A"})
		testkit.Equal(t, len(visualize.SnapshotOverlay(&failure.Snapshot{}).Render(tr)), 0, "empty-snap render")
	})

	t.Run("empty trace renders no markers even with non-empty snapshot", func(t *testing.T) {
		t.Parallel()
		snap := &failure.Snapshot{PerComponent: map[string]any{"A": 1}}
		testkit.Equal(t, len(visualize.SnapshotOverlay(snap).Render(trace.New())), 0, "empty-trace render")
	})

	t.Run("populated snapshot emits one marker per entry, attached to the last event", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{StartNs: 0, EndNs: 1, Method: "A"})
		last := tr.Record(trace.Event{StartNs: 1, EndNs: 2, Method: "B"})

		snap := &failure.Snapshot{
			PerComponent: map[string]any{"Beta": 1, "Alpha": 2},
			PerImpl:      map[string]any{"v1": 1},
			Custom:       map[string]any{"trace": "ok"},
		}
		markers := visualize.SnapshotOverlay(snap).Render(tr)
		testkit.Equal(t, len(markers), 4, "2 components + 1 impl + 1 custom = 4 markers")
		for _, m := range markers {
			testkit.Equal(t, m.EventID, last, "all markers attach to the last event")
			testkit.Equal(t, m.Layer, "snapshot", "layer")
		}

		styles := []string{markers[0].Style, markers[1].Style}
		testkit.True(t, styles[0] == "component" && styles[1] == "component", "components emitted first")
		testkit.True(t,
			strings.Contains(markers[0].Tooltip, "Alpha") &&
				strings.Contains(markers[1].Tooltip, "Beta"),
			"PerComponent keys emitted in sorted order")
		testkit.Equal(t, markers[2].Style, "impl", "impl style")
		testkit.Equal(t, markers[3].Style, "custom", "custom style")
	})
}

func TestEmitWithAllOverlays(t *testing.T) {
	t.Parallel()

	t.Run("integrates every overlay flavor without dropping markers", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		first := tr.Record(trace.Event{
			StartNs: 0, EndNs: 100, Method: "A", Component: "L",
			REQTags: []string{"REQ-1"},
		})
		tr.Record(trace.Event{
			StartNs: 100, EndNs: 200, Method: "B", Component: "L",
			Causality:    []trace.EventID{first},
			FaultContext: &trace.FaultContext{Affected: true, Active: []trace.FaultActivation{{Name: "Drop"}}},
			Metadata:     map[string]any{"replay_origin": "x"},
		})

		snap := &failure.Snapshot{PerComponent: map[string]any{"L": "ok", "M": "ok"}}

		tl := visualize.Timeline{
			Subject: "x", Generator: "model", Trace: tr,
			Overlays: []visualize.Overlay{
				visualize.FaultOverlay(),
				visualize.CausalityOverlay(),
				visualize.REQOverlay(),
				visualize.ReplayMarkerOverlay(),
				visualize.SnapshotOverlay(snap),
				visualize.DivergenceOverlay([]visualize.DivergenceMarker{
					{EventID: first, Lanes: map[string]any{"v1": 1, "v2": 2}, Diff: "differ"},
				}),
			},
		}

		var buf bytes.Buffer
		testkit.NoError(t, visualize.Emit(&buf, tl), "emit")
		out := buf.String()

		testkit.Assert(t, out).
			Contains("marker-fault", "fault marker rendered").
			Contains("marker-causality", "causality marker rendered").
			Contains("marker-req", "REQ marker rendered").
			Contains("marker-replay", "replay marker rendered").
			Contains("marker-snapshot", "snapshot marker rendered").
			Contains("marker-divergence", "divergence marker rendered")
	})

	t.Run("markers referencing unknown EventIDs are dropped silently", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{StartNs: 0, EndNs: 1, Method: "A"})
		tl := visualize.Timeline{
			Subject: "x", Generator: "model", Trace: tr,
			Overlays: []visualize.Overlay{
				visualize.DivergenceOverlay([]visualize.DivergenceMarker{
					{EventID: 999, Lanes: map[string]any{"v1": 1}, Diff: "phantom"},
				}),
			},
		}
		var buf bytes.Buffer
		testkit.NoError(t, visualize.Emit(&buf, tl), "emit")
		testkit.Assert(t, buf.String()).NotContains("phantom", "tooltip for unknown-ID marker is suppressed")
	})

	t.Run("causality marker without a known predecessor draws no edge", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		// Record only one event but tag it with causality referencing a phantom predecessor.
		tr.Record(trace.Event{StartNs: 0, EndNs: 1, Method: "A", Causality: []trace.EventID{999}})
		tl := visualize.Timeline{
			Subject: "x", Generator: "model", Trace: tr,
			Overlays: []visualize.Overlay{visualize.CausalityOverlay()},
		}
		var buf bytes.Buffer
		testkit.NoError(t, visualize.Emit(&buf, tl), "emit")
		testkit.Assert(t, buf.String()).NotContains("marker-causality\" d=", "no path drawn for missing predecessor")
	})

	t.Run("causality marker with empty predecessors metadata draws no edge", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		id := tr.Record(trace.Event{StartNs: 0, EndNs: 1, Method: "A"})
		tl := visualize.Timeline{
			Subject: "x", Generator: "model", Trace: tr,
			Overlays: []visualize.Overlay{
				rawOverlay("causality", id, map[string]any{"predecessors": []trace.EventID{}}),
			},
		}
		var buf bytes.Buffer
		testkit.NoError(t, visualize.Emit(&buf, tl), "emit")
		testkit.Assert(t, buf.String()).NotContains("marker-causality\" d=", "no path drawn for empty predecessors")
	})

	t.Run("causality marker without typed predecessors metadata draws no edge", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		id := tr.Record(trace.Event{StartNs: 0, EndNs: 1, Method: "A"})
		tl := visualize.Timeline{
			Subject: "x", Generator: "model", Trace: tr,
			Overlays: []visualize.Overlay{
				rawOverlay("causality", id, map[string]any{"predecessors": "not-a-slice"}),
			},
		}
		var buf bytes.Buffer
		testkit.NoError(t, visualize.Emit(&buf, tl), "emit")
		testkit.Assert(t, buf.String()).NotContains("marker-causality\" d=", "no path drawn for wrong-typed metadata")
	})
}

// rawOverlay returns a minimal Overlay producing a single marker with
// the supplied Layer/EventID/Metadata. Used for exercising
// fillCausalityEdge's defensive branches without going through the
// public CausalityOverlay constructor.
func rawOverlay(layer string, id trace.EventID, meta map[string]any) visualize.Overlay {
	return overlayFunc{layer: layer, id: id, meta: meta}
}

type overlayFunc struct {
	layer string
	id    trace.EventID
	meta  map[string]any
}

func (overlayFunc) Name() string { return "raw" }

func (o overlayFunc) Render(*trace.Trace) []visualize.Marker {
	return []visualize.Marker{{
		EventID:  o.id,
		Layer:    o.layer,
		Style:    "test",
		Tooltip:  "test",
		Metadata: o.meta,
	}}
}
