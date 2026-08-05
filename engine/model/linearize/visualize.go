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
// dir/linearizability-<seed>.html. The model runner calls this from
// the concurrent path so the consumer-supplied artifact dir collects
// every diagnostic artifact under one naming convention.
//
// seed names the run (sanitized test name or hex seed). An empty
// seed produces dir/linearizability.html. Returns the absolute path
// of the emitted file on success, or the empty string when no file
// was written (e.g., dir empty or Porcupine's visualization helper
// failed); failures are logged through tb.Logf so the surrounding
// failure report can still proceed.
func VisualizeOnFailure(tb testing.TB, m porcupine.Model, info porcupine.LinearizationInfo, dir, seed string) string {
	tb.Helper()
	if dir == "" {
		return ""
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		tb.Logf("VisualizeOnFailure: mkdir %s: %v", dir, err)
		return ""
	}
	name := "linearizability"
	if seed != "" {
		name = "linearizability-" + seed
	}
	path := filepath.Join(dir, name+".html")
	if err := porcupine.VisualizePath(m, info, path); err != nil {
		tb.Logf("VisualizeOnFailure: VisualizePath: %v", err)
		return ""
	}
	return path
}
