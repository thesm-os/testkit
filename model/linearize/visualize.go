// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anishathalye/porcupine"
)

// VisualizeOnFailure writes a Porcupine HTML visualization to
// dir/<seed>.html when the linearizability check returns Illegal.
// Used by the model runner's failure-classification path: on an
// invariant-class failure the consumer-supplied dir collects every
// diagnostic artifact.
//
// The file name is derived from seed; an empty seed defaults to
// "linearizability.html". Returns the absolute path of the emitted
// file on success, or the empty string when no file was written
// (e.g., dir empty or Porcupine's visualization helper failed).
//
// VisualizeOnFailure is a no-op when info.Result != Illegal.
func VisualizeOnFailure(tb testing.TB, m porcupine.Model, info porcupine.LinearizationInfo, dir, seed string) string {
	tb.Helper()
	if dir == "" {
		return ""
	}
	if seed == "" {
		seed = "linearizability"
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		tb.Logf("VisualizeOnFailure: mkdir %s: %v", dir, err)
		return ""
	}
	path := filepath.Join(dir, seed+".html")
	if err := porcupine.VisualizePath(m, info, path); err != nil {
		tb.Logf("VisualizeOnFailure: VisualizePath: %v", err)
		return ""
	}
	return path
}
