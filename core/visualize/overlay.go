// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package visualize

import "go.thesmos.sh/testkit/core/trace"

// Layer names used by built-in overlays. Custom overlays may define
// their own; the renderer dispatches on the string value, so any
// layer not listed here renders with a generic style or no special
// treatment depending on how the template is extended.
const (
	LayerFault      = "fault"
	LayerDivergence = "divergence"
	LayerCausality  = "causality"
	LayerREQ        = "req"
	LayerReplay     = "replay"
	LayerSnapshot   = "snapshot"
)

// Overlay produces visual annotations layered on the timeline. Each
// generator contributes its own overlay flavor: chaos adds fault
// windows; diff-rollout adds divergence markers; sim adds causality
// arrows; replay adds input-event markers; failure capture adds
// snapshot indicators. Consumer-defined overlays implement this
// interface and pass through [Timeline.Overlays].
type Overlay interface {
	// Name identifies the overlay layer. Each overlay's markers
	// share this name so the renderer can group them under one
	// CSS class and one toggle in the legend.
	Name() string

	// Render produces markers for the given trace. Called once per
	// emit. The trace is read-only; overlays must not mutate it.
	Render(t *trace.Trace) []Marker
}

// Marker annotates a specific event on the timeline. Markers are
// rendered as visual decorations on top of the event blocks: fault
// markers as colored bands, causality arrows as SVG paths, REQ
// tags as small labels, etc.
type Marker struct {
	// EventID identifies the event the marker attaches to. Marker
	// rendering looks up the event's position by ID; markers
	// referencing unknown IDs are skipped silently.
	EventID trace.EventID

	// Layer is the overlay name; rendered as a CSS class so
	// per-layer styling and toggling is possible.
	Layer string

	// Style is an optional CSS-class suffix for per-marker
	// variation within a layer (e.g., "fault-network" vs
	// "fault-clock"). Combined with Layer to form the full class:
	// "marker-<Layer>-<Style>".
	Style string

	// Tooltip is the text shown on hover via SVG `<title>`. Plain
	// text only; HTML escaping is the renderer's job.
	Tooltip string

	// Metadata carries overlay-specific data the renderer ignores.
	// Useful for downstream tools that walk the marker list
	// (e.g., a CI bot extracting fault names).
	Metadata map[string]any
}

// DivergenceMarker is the input shape for [DivergenceOverlay]. A
// diff-rollout run records one entry per detected divergence: which
// event, which lanes diverged, what the diff was.
type DivergenceMarker struct {
	// EventID is the event where divergence first appeared.
	EventID trace.EventID

	// Lanes maps an impl name to the value it produced for the
	// diverging method. The renderer surfaces these in the marker
	// tooltip.
	Lanes map[string]any

	// Diff is a human-readable diff between the lanes.
	Diff string
}
