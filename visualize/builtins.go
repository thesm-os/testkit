// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package visualize

import (
	"fmt"
	"sort"
	"strings"

	"go.thesmos.sh/testkit/failure"
	"go.thesmos.sh/testkit/trace"
)

// FaultOverlay marks every event whose [trace.Event.FaultContext]
// reports the call was affected by an active fault. Used by chaos:
// the engine populates FaultContext per-call; the overlay surfaces
// the fault-affected events so a reader can trace which calls
// participated in a fault window.
func FaultOverlay() Overlay { return faultOverlay{} }

type faultOverlay struct{}

func (faultOverlay) Name() string { return LayerFault }

func (faultOverlay) Render(t *trace.Trace) []Marker {
	if t == nil {
		return nil
	}
	var out []Marker
	for _, e := range t.Snapshot() {
		if e.FaultContext == nil || !e.FaultContext.Affected {
			continue
		}
		out = append(out, Marker{
			EventID: e.ID,
			Layer:   LayerFault,
			Style:   styleFromFaultNames(e.FaultContext.Active),
			Tooltip: faultTooltip(e),
		})
	}
	return out
}

func styleFromFaultNames(activations []trace.FaultActivation) string {
	if len(activations) == 0 {
		return "generic"
	}
	// Use the first fault's name as the style suffix; multi-fault
	// events still get a single CSS class for the dominant fault.
	return strings.ToLower(strings.ReplaceAll(activations[0].Name, " ", "-"))
}

func faultTooltip(e trace.Event) string {
	if e.FaultContext == nil || len(e.FaultContext.Active) == 0 {
		return "fault active"
	}
	names := make([]string, 0, len(e.FaultContext.Active))
	for _, a := range e.FaultContext.Active {
		if a.Component != "" {
			names = append(names, a.Name+"@"+a.Component)
		} else {
			names = append(names, a.Name)
		}
	}
	return strings.Join(names, ", ")
}

// DivergenceOverlay emits one marker per provided
// [DivergenceMarker]. Used by diff-rollout: the runner records one
// entry per detected divergence; the overlay surfaces them on the
// timeline at the offending event with a tooltip naming the diverging
// lanes.
func DivergenceOverlay(divergences []DivergenceMarker) Overlay {
	return divergenceOverlay{divs: divergences}
}

type divergenceOverlay struct {
	divs []DivergenceMarker
}

func (divergenceOverlay) Name() string { return LayerDivergence }

func (o divergenceOverlay) Render(_ *trace.Trace) []Marker {
	out := make([]Marker, 0, len(o.divs))
	for _, d := range o.divs {
		laneNames := make([]string, 0, len(d.Lanes))
		for name := range d.Lanes {
			laneNames = append(laneNames, name)
		}
		sort.Strings(laneNames)
		tooltip := "divergence: " + strings.Join(laneNames, " vs ")
		if d.Diff != "" {
			tooltip += " — " + d.Diff
		}
		out = append(out, Marker{
			EventID:  d.EventID,
			Layer:    LayerDivergence,
			Style:    "first",
			Tooltip:  tooltip,
			Metadata: map[string]any{"lanes": laneNames},
		})
	}
	return out
}

// CausalityOverlay emits a marker for every event whose
// [trace.Event.Causality] list is non-empty. Used by sim: the engine
// records cross-component happens-before predecessors; the overlay
// surfaces them so a reader can follow causality chains.
func CausalityOverlay() Overlay { return causalityOverlay{} }

type causalityOverlay struct{}

func (causalityOverlay) Name() string { return LayerCausality }

func (causalityOverlay) Render(t *trace.Trace) []Marker {
	if t == nil {
		return nil
	}
	var out []Marker
	for _, e := range t.Snapshot() {
		if len(e.Causality) == 0 {
			continue
		}
		ids := make([]string, 0, len(e.Causality))
		for _, p := range e.Causality {
			ids = append(ids, fmt.Sprintf("%d", p))
		}
		out = append(out, Marker{
			EventID: e.ID,
			Layer:   LayerCausality,
			Style:   "predecessor",
			Tooltip: "after: " + strings.Join(ids, ", "),
			Metadata: map[string]any{
				"predecessors": e.Causality,
			},
		})
	}
	return out
}

// REQOverlay emits a marker for every event tagged with one or more
// `//testkit:req REQ-...` requirement IDs. Used by model and sim:
// the generator threads REQ tags through into emitted laws and
// trace events; the overlay surfaces them so a reader can match
// observations to requirements.
func REQOverlay() Overlay { return reqOverlay{} }

type reqOverlay struct{}

func (reqOverlay) Name() string { return LayerREQ }

func (reqOverlay) Render(t *trace.Trace) []Marker {
	if t == nil {
		return nil
	}
	var out []Marker
	for _, e := range t.Snapshot() {
		if len(e.REQTags) == 0 {
			continue
		}
		out = append(out, Marker{
			EventID:  e.ID,
			Layer:    LayerREQ,
			Style:    "tag",
			Tooltip:  "REQ: " + strings.Join(e.REQTags, ", "),
			Metadata: map[string]any{"tags": e.REQTags},
		})
	}
	return out
}

// ReplayMarkerOverlay emits a marker for every event tagged as
// originating from a replay input (events the replay runner drove
// through the SUT from a recorded trace). Used by replay: the
// runner sets `Metadata["replay_origin"]` to the source event ID;
// the overlay surfaces these so a reader can cross-reference back
// to the input trace.
func ReplayMarkerOverlay() Overlay { return replayOverlay{} }

type replayOverlay struct{}

func (replayOverlay) Name() string { return LayerReplay }

func (replayOverlay) Render(t *trace.Trace) []Marker {
	if t == nil {
		return nil
	}
	var out []Marker
	for _, e := range t.Snapshot() {
		origin, ok := e.Metadata["replay_origin"]
		if !ok {
			continue
		}
		out = append(out, Marker{
			EventID:  e.ID,
			Layer:    LayerReplay,
			Style:    "input",
			Tooltip:  fmt.Sprintf("replay_origin: %v", origin),
			Metadata: map[string]any{"origin": origin},
		})
	}
	return out
}

// SnapshotOverlay emits markers for the entries in a captured
// [failure.Snapshot]. Used by sim, chaos, and diff-rollout when
// capture-on-failure populates per-component or per-impl state.
// Each entry is rendered as a marker attached to the last event in
// the trace (the failure tick), with a tooltip showing the entry's
// key. The renderer uses the marker's Style to distinguish per-
// component from per-impl entries.
func SnapshotOverlay(snap *failure.Snapshot) Overlay {
	return snapshotOverlay{snap: snap}
}

type snapshotOverlay struct {
	snap *failure.Snapshot
}

func (snapshotOverlay) Name() string { return LayerSnapshot }

func (o snapshotOverlay) Render(t *trace.Trace) []Marker {
	if t == nil || o.snap == nil || o.snap.IsEmpty() {
		return nil
	}
	events := t.Snapshot()
	if len(events) == 0 {
		return nil
	}
	last := events[len(events)-1].ID

	var out []Marker
	for _, k := range sortedKeys(o.snap.PerComponent) {
		out = append(out, Marker{
			EventID:  last,
			Layer:    LayerSnapshot,
			Style:    "component",
			Tooltip:  "component: " + k,
			Metadata: map[string]any{"component": k},
		})
	}
	for _, k := range sortedKeys(o.snap.PerImpl) {
		out = append(out, Marker{
			EventID:  last,
			Layer:    LayerSnapshot,
			Style:    "impl",
			Tooltip:  "impl: " + k,
			Metadata: map[string]any{"impl": k},
		})
	}
	for _, k := range sortedKeys(o.snap.Custom) {
		out = append(out, Marker{
			EventID:  last,
			Layer:    LayerSnapshot,
			Style:    "custom",
			Tooltip:  "custom: " + k,
			Metadata: map[string]any{"key": k},
		})
	}
	return out
}

// sortedKeys returns m's keys in sorted order. Used by overlays
// whose output must be deterministic across runs (map iteration
// would otherwise break the byte-identical-rendering contract).
func sortedKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
