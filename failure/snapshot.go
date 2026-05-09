// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package failure

// Snapshot holds per-component or per-impl state captured at
// failure time. Populated by sim's capture-on-failure (one entry
// per wrapped component in [Snapshot.PerComponent]), by
// diff-rollout's divergence reporter (one entry per impl in
// [Snapshot.PerImpl]), and by any layer that wants generator-
// specific state via [Snapshot.Custom].
//
// Values are type-erased; consumers JSON-marshal what they store.
// Non-JSON-marshalable values cause the [Failure] envelope's
// MarshalJSON to fail rather than silently drop fields.
type Snapshot struct {
	// PerComponent maps a wrapped-component name (e.g., "Ledger",
	// "Scheduler") to a state value. Sim populates one entry per
	// component from the engine's component registry at failure tick.
	PerComponent map[string]any `json:"per_component,omitempty"`

	// PerImpl maps a NamedFactory's Name (e.g., "v1", "v2") to a
	// state snapshot for that impl. Diff-rollout populates one
	// entry per registered impl when divergence is detected.
	PerImpl map[string]any `json:"per_impl,omitempty"`

	// Custom carries arbitrary key/value pairs for layer-specific
	// state that doesn't fit per-component or per-impl. Replay
	// stores trace-source provenance here; chaos stores the
	// load-bearing-fault extraction result.
	Custom map[string]any `json:"custom,omitempty"`
}

// IsEmpty reports whether the snapshot has no entries in any of
// its three maps. The CI ingestion path uses this to skip emitting
// snapshot artifacts when no state was captured.
func (s *Snapshot) IsEmpty() bool {
	if s == nil {
		return true
	}
	return len(s.PerComponent) == 0 && len(s.PerImpl) == 0 && len(s.Custom) == 0
}
