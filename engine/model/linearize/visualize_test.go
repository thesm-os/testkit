// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anishathalye/porcupine"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/linearize"
)

func TestVisualizeOnFailure(t *testing.T) {
	t.Parallel()

	t.Run("empty dir returns empty path without writing", func(t *testing.T) {
		t.Parallel()
		_, info := porcupine.CheckOperationsVerbose(linearize.Counter(), nil, 0)
		got := linearize.VisualizeOnFailure(t, linearize.Counter(), info, "", "seed")
		testkit.Equal(t, got, "", "empty dir → empty path")
	})

	t.Run("populated dir writes the HTML at <dir>/linearizability-<seed>.html", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		m := linearize.Counter()
		// One linearizable op produces a valid info; the helper writes
		// the Porcupine HTML regardless of legality.
		history := []porcupine.Operation{
			opIO(0, "Inc", nil, int64(1)),
		}
		_, info := porcupine.CheckOperationsVerbose(m, history, 0)
		got := linearize.VisualizeOnFailure(t, m, info, dir, "abc")
		testkit.Equal(t, got, filepath.Join(dir, "linearizability-abc.html"), "path returned")
		if _, err := os.Stat(got); err != nil {
			t.Fatalf("expected file: %v", err)
		}
	})

	t.Run("empty seed defaults to linearizability.html", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		m := linearize.Counter()
		history := []porcupine.Operation{
			opIO(0, "Inc", nil, int64(1)),
		}
		_, info := porcupine.CheckOperationsVerbose(m, history, 0)
		got := linearize.VisualizeOnFailure(t, m, info, dir, "")
		testkit.Equal(t, got, filepath.Join(dir, "linearizability.html"), "default seed name")
	})
}
