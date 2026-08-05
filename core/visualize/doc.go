// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package visualize emits a self-contained HTML timeline for one
// run of any generator. One [Timeline] per failure or per
// post-run diagnostic; consumers add overlays for fault windows,
// divergence points, causality arrows, REQ tags, replay markers,
// per-component snapshots.
//
// The output is one HTML document with embedded CSS — no external
// dependencies, no JavaScript. Opens in any browser. Tooltips are
// rendered via SVG `<title>` elements; an event-detail table sits
// alongside the timeline for full-fidelity reading.
//
// Overlays are pluggable. Sim contributes [CausalityOverlay] and
// [REQOverlay]; chaos contributes [FaultOverlay]; diff-rollout
// contributes [DivergenceOverlay]; replay contributes
// [ReplayMarkerOverlay]; failure capture contributes
// [SnapshotOverlay]. Custom overlays implement the [Overlay]
// interface; the unified renderer composes the resulting markers
// into one timeline.
//
// Determinism: emit is byte-identical for byte-identical inputs.
// Map iteration order is normalized; no time.Now() calls; no
// random IDs. The same run on the same seed produces a diffable
// HTML artifact across reruns.
package visualize
