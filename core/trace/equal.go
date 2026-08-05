// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package trace

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// EqualForDeterminism compares two Traces under the determinism
// cross-validation contract: same seed → same trace. Engine-relative
// fields ([Event.Tick], [Event.StartNs], [Event.EndNs],
// [Event.Goroutine], [Event.ClientID], [Event.Causality]) ARE
// compared. Wall-clock-only fields are not present at the Event
// level — the file-header `captured_at` is a Trace-level concern
// the JSON layer strips before comparison.
//
// Used by Phase 0+ acceptance gates and by the model runner's
// `WithDeterminismCheck()` option to verify two runs at the same
// seed produce byte-identical traces. Returns true when the traces
// are determinism-equivalent.
//
// Internally uses go-cmp; consumers needing the diff should call
// [DiffForDeterminism] instead.
func EqualForDeterminism(a, b *Trace) bool {
	return DiffForDeterminism(a, b) == ""
}

// DiffForDeterminism returns a structured diff between two Traces
// under the determinism contract, or the empty string when they are
// equivalent. Used by the determinism-check gate and by the
// classified-Failure reporter to render trace divergences.
func DiffForDeterminism(a, b *Trace) string {
	left := a.Snapshot()
	right := b.Snapshot()
	return cmp.Diff(left, right, deterministicCmpOpts...)
}

// deterministicCmpOpts is the option set DiffForDeterminism uses.
// Today it permits unexported fields nowhere; every relevant Event
// field is exported. Fields that should be excluded from comparison
// (e.g., raw map iteration order in [Event.Metadata]) are filtered
// here when needed.
var deterministicCmpOpts = []cmp.Option{
	// Treat nil and empty slices as equal — empty Causality vs nil
	// Causality produces the same observable behavior and trace
	// JSON marshaling can normalize either way.
	cmpopts.EquateEmpty(),
	// Errors in events compare by message text. Two distinct error
	// values with identical messages are determinism-equivalent
	// because runtime trace equality cares about observable
	// signaling, not pointer identity.
	cmp.Comparer(errorMessageEqual),
}

func errorMessageEqual(a, b error) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Error() == b.Error()
}
