// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package visualize

import (
	"go.thesmos.sh/testkit/core/trace"
)

// Timeline carries one run's data into [Emit]. The Trace is the
// event source; overlays add layered annotations; Style controls
// rendering. Subject / Generator / Seed populate the page header
// and CI-artifact filename.
type Timeline struct {
	// Subject is the interface or subsystem name displayed in the
	// header (e.g., "basic.Store", "ledger.Subsystem").
	Subject string

	// Generator is the producing generator's name ("model" / "sim"
	// / "chaos" / "diff-rollout" / "replay") for the header.
	Generator string

	// Seed is the run seed displayed in the header. Rendered as
	// 0x-prefixed hex so reruns can copy-paste the value.
	Seed int64

	// Trace is the event source. Each event becomes a block on
	// the timeline, grouped into lanes by [trace.Event.Component]
	// (or by Goroutine when no Component is populated, the
	// per-interface model case).
	Trace *trace.Trace

	// Overlays add layered markers on top of the timeline.
	// Built-in overlays cover faults, divergence, causality, REQ
	// tags, replay markers, and snapshots; consumers register
	// custom overlays via the [Overlay] interface.
	Overlays []Overlay

	// Style controls visual presentation. Zero value applies the
	// light theme with default colors; consumers override the
	// theme or per-component colors selectively.
	Style Style
}

// Style controls visual presentation. The zero value is usable
// (light theme, default colors); consumers override fields as
// needed.
type Style struct {
	// Theme is "light" or "dark". Empty defaults to "light".
	Theme string

	// ComponentColors maps a component name to a CSS color
	// (#hex, rgb(), or named). Components without an entry get a
	// deterministic color from a fixed palette indexed by the
	// component's name hash.
	ComponentColors map[string]string

	// Title overrides the page title; empty uses
	// "<Subject> — <Generator> timeline".
	Title string
}

// theme returns the configured theme or the default.
func (s Style) theme() string {
	if s.Theme == "" {
		return "light"
	}
	return s.Theme
}

// title returns the configured title or a default derived from
// Timeline metadata.
func (t Timeline) title() string {
	if t.Style.Title != "" {
		return t.Style.Title
	}
	return t.Subject + " — " + t.Generator + " timeline"
}
