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

// The visualiser runs inside a failure path, so a filesystem problem must
// degrade to "no artifact" rather than compounding the failure it is trying to
// document.
func TestVisualizeOnFailureWriteFailures(t *testing.T) {
	t.Parallel()

	t.Run("an unmakeable directory logs and yields no path", func(t *testing.T) {
		t.Parallel()
		blocker := filepath.Join(t.TempDir(), "blocker")
		if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
			t.Fatalf("setup: %v", err)
		}
		ft := testkit.NewFailableTB()
		m := linearize.Counter()
		_, info := porcupine.CheckOperationsVerbose(m, nil, 0)

		got := linearize.VisualizeOnFailure(ft, m, info, filepath.Join(blocker, "sub"), "seed")
		testkit.Equal(t, got, "", "a failed mkdir yields no artifact path")
		if len(ft.Logs()) == 0 {
			t.Fatal("the failure must be logged, not swallowed")
		}
		if ft.Failed() {
			t.Fatal("an artifact failure must not fail the test it documents")
		}
	})

	// A path that cannot be written — here a directory standing where the HTML
	// file belongs — exercises the second failure arm, past mkdir.
	t.Run("an unwritable target logs and yields no path", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "linearizability-seed.html"), 0o750); err != nil {
			t.Fatalf("setup: %v", err)
		}
		ft := testkit.NewFailableTB()
		m := linearize.Counter()
		_, info := porcupine.CheckOperationsVerbose(m, nil, 0)

		got := linearize.VisualizeOnFailure(ft, m, info, dir, "seed")
		testkit.Equal(t, got, "", "an unwritable target yields no artifact path")
		if len(ft.Logs()) == 0 {
			t.Fatal("the failure must be logged, not swallowed")
		}
	})
}
