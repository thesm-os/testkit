// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal tests for the artifact writer behind the concurrent runner. It is
// unexported because consumers reach the same behaviour through
// [linearize.VisualizeOnFailure]; the runner's copy still has to degrade
// safely, and only an internal test can drive it directly.
package model

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anishathalye/porcupine"
	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/trace"
	"go.thesmos.sh/testkit/engine/model/law"
)

// trivialModel stands in for a real linearizability model. The visualizer
// only needs a model to render against; linearize cannot be imported here
// because it depends on this package.
func trivialModel() porcupine.Model {
	return porcupine.Model{
		Init: func() any { return 0 },
		Step: func(state, _, _ any) (bool, any) { return true, state },
	}
}

func TestWriteVisualization(t *testing.T) {
	t.Parallel()

	// An empty info is enough: the writer's contract is about where the file
	// lands and what happens when it cannot, not about what it contains.
	_, info := porcupine.CheckOperationsVerbose(trivialModel(), nil, 0)

	t.Run("an empty artifact dir writes nothing", func(t *testing.T) {
		t.Parallel()
		ft := testkit.NewFailableTB().WithName("nodir")
		if got := writeVisualization(ft, trivialModel(), info, ""); got != "" {
			t.Fatalf("no artifact dir means no artifact, got %s", got)
		}
	})

	t.Run("a populated dir yields linearizability-<seed>.html", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		ft := testkit.NewFailableTB().WithName("run/one")
		got := writeVisualization(ft, trivialModel(), info, dir)
		if got == "" {
			t.Fatal("a writable dir must produce an artifact")
		}
		// The test name carries a slash; the filename must not.
		if base := filepath.Base(got); base != "linearizability-run_one.html" {
			t.Fatalf("unexpected artifact name %q", base)
		}
		if _, err := os.Stat(got); err != nil {
			t.Fatalf("the artifact must exist: %v", err)
		}
	})

	// The writer runs inside a failure path, so a filesystem problem must
	// degrade to "no artifact" rather than compounding the failure.
	t.Run("an unmakeable dir logs and yields no path", func(t *testing.T) {
		t.Parallel()
		blocker := filepath.Join(t.TempDir(), "blocker")
		if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
			t.Fatalf("setup: %v", err)
		}
		ft := testkit.NewFailableTB().WithName("blocked")
		if got := writeVisualization(ft, trivialModel(), info, filepath.Join(blocker, "sub")); got != "" {
			t.Fatalf("a failed mkdir must yield no path, got %s", got)
		}
		if len(ft.Logs()) == 0 {
			t.Fatal("the failure must be logged, not swallowed")
		}
		if ft.Failed() {
			t.Fatal("an artifact failure must not fail the test it documents")
		}
	})

	t.Run("an unwritable target logs and yields no path", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		// A directory standing where the HTML belongs gets past mkdir and
		// fails at the write.
		if err := os.MkdirAll(filepath.Join(dir, "linearizability-taken.html"), 0o750); err != nil {
			t.Fatalf("setup: %v", err)
		}
		ft := testkit.NewFailableTB().WithName("taken")
		if got := writeVisualization(ft, trivialModel(), info, dir); got != "" {
			t.Fatalf("an unwritable target must yield no path, got %s", got)
		}
		if len(ft.Logs()) == 0 {
			t.Fatal("the failure must be logged, not swallowed")
		}
	})
}

// TestTraceLawWalkSkipsStepBoundaryLaws pins the walk's defensive arms: a
// registry holding only step-boundary laws scans no trace, and the walk
// steps over one rather than binding what has no BindTrace. The dispatch
// guard rejects that mix before any real run reaches here — these arms are
// what keeps a future dispatch change from corrupting the walk silently.
func TestTraceLawWalkSkipsStepBoundaryLaws(t *testing.T) {
	t.Parallel()

	r := NewRegistry[int]()
	r.Add(law.CountEqualsReference[int, int]{
		Count: func(_ *rapid.T, n int) (int, error) { return n, nil },
	})
	if hasTraceLaws(r) {
		t.Fatal("a step-boundary law scans no trace")
	}

	rapid.Check(t, func(rt *rapid.T) {
		checkTraceLaws(rt, Config[int]{Laws: r}, &trace.Trace{}, 1, 1)
	})
	if r.ran[law.CountEqualsReference[int, int]{}.ID()] != 0 {
		t.Fatal("the walk must step over a law it cannot bind a trace to")
	}
}
