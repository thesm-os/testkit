// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.thesmos.sh/testkit/core/trace"
)

//nolint:revive // test error strings are diagnostic, not user-facing
func TestFormatFailure(t *testing.T) {
	t.Parallel()

	t.Run("semantic failure without trace", func(t *testing.T) {
		t.Parallel()
		f := &Failure{
			Kind:         FailureSemantic,
			StepRan:      StepID{WorkerID: -1, Index: 2},
			StepReported: StepID{WorkerID: -1, Index: 2},
			Err:          errors.New("Get(\"k\"): SUT err=not found, ref err=<nil>"),
		}
		out := formatFailure(f)
		if !strings.Contains(out, "[semantic]") {
			t.Fatalf("missing kind tag: %s", out)
		}
		if !strings.Contains(out, "at step 2") {
			t.Fatalf("missing step: %s", out)
		}
		// Semantic failures have no trace.
		if strings.Contains(out, "[step") {
			t.Fatalf("semantic should not have trace dump: %s", out)
		}
	})

	t.Run("invariant failure with trace and REQID", func(t *testing.T) {
		t.Parallel()
		f := &Failure{
			Kind:         FailureInvariant,
			LawID:        "AUTO-READ-AFTER-WRITE",
			REQID:        "REQ-LEDGER-014",
			StepRan:      StepID{WorkerID: -1, Index: 5},
			StepReported: StepID{WorkerID: -1, Index: 5},
			Err:          errors.New("key \"x\": SUT/ref disagree"),
			Trace: []trace.Event{
				{ClientID: -1, Method: "Put", Inputs: []any{"item-a"}, Output: nil},
				{ClientID: -1, Method: "Put", Inputs: []any{"item-b"}, Output: nil},
				{ClientID: -1, Method: "Get", Inputs: []any{"x"}, Output: "old"},
			},
		}
		out := formatFailure(f)
		if !strings.Contains(out, "[REQ-LEDGER-014 invariant]") {
			t.Fatalf("missing REQID+kind: %s", out)
		}
		if !strings.Contains(out, "AUTO-READ-AFTER-WRITE") {
			t.Fatalf("missing law ID: %s", out)
		}
		if !strings.Contains(out, "[step 0] Put") {
			t.Fatalf("missing trace step 0: %s", out)
		}
		if !strings.Contains(out, "[step 2]") {
			t.Fatalf("missing trace step 2: %s", out)
		}
	})

	t.Run("structural failure with trace annotation", func(t *testing.T) {
		t.Parallel()
		f := &Failure{
			Kind:         FailureStructural,
			StepRan:      StepID{WorkerID: -1, Index: 1},
			StepReported: StepID{WorkerID: -1, Index: 1},
			Err:          errors.New("Verify: SUT err=chain integrity, ref err=<nil>"),
			Trace: []trace.Event{
				{ClientID: -1, Method: "Append", Inputs: []any{"entry-1"}, Output: nil},
				{ClientID: -1, Method: "Verify", Inputs: nil, Output: nil, Err: errors.New("chain integrity")},
			},
		}
		out := formatFailure(f)
		if !strings.Contains(out, "[structural]") {
			t.Fatalf("missing kind: %s", out)
		}
		if !strings.Contains(out, "<-- FAILED") {
			t.Fatalf("missing failure annotation: %s", out)
		}
	})

	t.Run("liveness failure without trace", func(t *testing.T) {
		t.Parallel()
		f := &Failure{
			Kind:          FailureLiveness,
			LawID:         "AUTO-NO-GOROUTINE-LEAKS",
			StepRan:       StepID{WorkerID: -1, Index: 0},
			StepReported:  StepID{WorkerID: -1, Index: 0},
			Err:           errors.New("2 goroutine(s) leaked"),
			ArtifactPaths: []string{"viz: .testkit/artifacts/TestStore-goroutines.txt"},
		}
		out := formatFailure(f)
		if !strings.Contains(out, "[liveness]") {
			t.Fatalf("missing kind: %s", out)
		}
		if !strings.Contains(out, "goroutines.txt") {
			t.Fatalf("missing artifact path: %s", out)
		}
		// Liveness has no trace (stacks instead).
		if strings.Contains(out, "[step") {
			t.Fatalf("liveness should not have trace dump: %s", out)
		}
	})

	t.Run("unclassified failure with trace", func(t *testing.T) {
		t.Parallel()
		f := &Failure{
			Kind:         FailureUnclassified,
			StepRan:      StepID{WorkerID: -1, Index: 3},
			StepReported: StepID{WorkerID: -1, Index: 3},
			Err:          errors.New("custom check failed"),
			Trace: []trace.Event{
				{ClientID: -1, Method: "Put", Inputs: []any{"v"}, Output: nil},
			},
		}
		out := formatFailure(f)
		if !strings.Contains(out, "[unclassified]") {
			t.Fatalf("missing kind: %s", out)
		}
		if !strings.Contains(out, "[step 0] Put") {
			t.Fatalf("missing trace: %s", out)
		}
	})

	t.Run("concurrent format with worker prefix", func(t *testing.T) {
		t.Parallel()
		f := &Failure{
			Kind:         FailureStructural,
			StepRan:      StepID{WorkerID: 2, Index: 3},
			StepReported: StepID{WorkerID: 2, Index: 3},
			Err:          errors.New("history is not linearizable"),
		}
		out := formatFailure(f)
		if !strings.Contains(out, "w2 op3") {
			t.Fatalf("missing worker/op prefix: %s", out)
		}
	})
}

func TestTruncateValue(t *testing.T) {
	t.Parallel()

	t.Run("short string unchanged", func(t *testing.T) {
		t.Parallel()
		got := truncateValue("hello")
		if got != `"hello"` {
			t.Fatalf("expected quoted string, got %s", got)
		}
	})

	t.Run("long string truncated", func(t *testing.T) {
		t.Parallel()
		long := strings.Repeat("x", 50)
		got := truncateValue(long)
		if !strings.Contains(got, "...") {
			t.Fatalf("expected truncation marker: %s", got)
		}
		_ = got // length checked by truncation marker above
	})

	t.Run("nil returns nil marker", func(t *testing.T) {
		t.Parallel()
		got := truncateValue(nil)
		if got != "<nil>" {
			t.Fatalf("expected <nil>, got %s", got)
		}
	})

	t.Run("slice of any", func(t *testing.T) {
		t.Parallel()
		got := truncateValue([]any{"a", "b"})
		if !strings.Contains(got, "a") || !strings.Contains(got, "b") {
			t.Fatalf("expected both items: %s", got)
		}
	})
}

func TestSanitizeForFilename(t *testing.T) {
	t.Parallel()

	t.Run("replaces slashes and special chars", func(t *testing.T) {
		t.Parallel()
		got := sanitizeForFilename("Test/Sub: foo*bar?")
		if strings.ContainsAny(got, "/: *?") {
			t.Fatalf("should not contain special chars: %s", got)
		}
	})

	t.Run("preserves valid chars", func(t *testing.T) {
		t.Parallel()
		got := sanitizeForFilename("TestStore.Model-run")
		if got != "TestStore.Model-run" {
			t.Fatalf("expected no change, got %s", got)
		}
	})

	t.Run("truncates to 200 chars", func(t *testing.T) {
		t.Parallel()
		long := strings.Repeat("a", 300)
		got := sanitizeForFilename(long)
		if len(got) != 200 {
			t.Fatalf("expected 200 chars, got %d", len(got))
		}
	})
}

// truncateValue renders an action's inputs and outputs into a failure report,
// so it must stay bounded for every type it might be handed — an untruncated
// 10KB payload turns a diagnostic into a wall of text.
func TestTruncateValueByType(t *testing.T) {
	t.Parallel()

	t.Run("short byte slices render as hex", func(t *testing.T) {
		t.Parallel()
		got := truncateValue([]byte{0xde, 0xad})
		if got != "dead" {
			t.Fatalf("expected hex encoding, got %s", got)
		}
	})

	t.Run("long byte slices report their size and truncate", func(t *testing.T) {
		t.Parallel()
		got := truncateValue(make([]byte, 200))
		if !strings.Contains(got, "200 bytes") {
			t.Fatalf("the size must be reported, got %s", got)
		}
		if !strings.Contains(got, "...") {
			t.Fatalf("the payload must be truncated, got %s", got)
		}
	})

	t.Run("short errors render their message", func(t *testing.T) {
		t.Parallel()
		got := truncateValue(errors.New("boom"))
		if got != "boom" {
			t.Fatalf("expected the bare message, got %s", got)
		}
	})

	t.Run("long errors truncate", func(t *testing.T) {
		t.Parallel()
		got := truncateValue(errors.New(strings.Repeat("e", 50)))
		if !strings.HasSuffix(got, "...") {
			t.Fatalf("a long error must be truncated, got %s", got)
		}
	})

	// A single-element slice unwraps rather than rendering as a one-item list,
	// because most actions take exactly one argument and "[x]" reads worse
	// than "x" in a report.
	t.Run("slices of one unwrap, empty slices render as brackets", func(t *testing.T) {
		t.Parallel()
		if got := truncateValue([]any{}); got != "[]" {
			t.Fatalf("expected empty brackets, got %s", got)
		}
		if got := truncateValue([]any{"only"}); got != `"only"` {
			t.Fatalf("a single element must unwrap, got %s", got)
		}
	})

	t.Run("other types render with %+v and truncate when long", func(t *testing.T) {
		t.Parallel()
		type small struct{ A int }
		if got := truncateValue(small{A: 1}); !strings.Contains(got, "A:1") {
			t.Fatalf("structs render field-wise, got %s", got)
		}
		type big struct{ S string }
		got := truncateValue(big{S: strings.Repeat("y", 200)})
		if !strings.HasSuffix(got, "...") {
			t.Fatalf("a long struct rendering must be truncated, got %s", got)
		}
	})
}

// The artifact directory anchors on the project config rather than go.mod:
// in a multi-module workspace every sub-module has a go.mod, so anchoring on
// that would scatter artifacts one directory per module.
func TestResolveArtifactDir(t *testing.T) {
	t.Parallel()

	t.Run("an explicit override wins", func(t *testing.T) {
		t.Parallel()
		if got := ResolveArtifactDir("/tmp/custom"); got != "/tmp/custom" {
			t.Fatalf("an override must be returned verbatim, got %s", got)
		}
	})

	t.Run("without an override it resolves under a project root", func(t *testing.T) {
		t.Parallel()
		got := ResolveArtifactDir("")
		if got == "" {
			t.Fatal("a default artifact dir must always resolve")
		}
		if !strings.HasSuffix(got, defaultArtifactDir) {
			t.Fatalf("the default suffix must be preserved, got %s", got)
		}
	})
}

// Outside any module the walk finds no anchor at all, and the resolver still
// has to name somewhere to write — a relative path under the working
// directory. t.Chdir forbids a parallel test, so this stands on its own.
//
//nolint:paralleltest // t.Chdir cannot be used from a parallel test
func TestResolveArtifactDirOutsideAnyProject(t *testing.T) {
	t.Chdir(t.TempDir())
	if got := ResolveArtifactDir(""); got != defaultArtifactDir {
		t.Fatalf("expected the bare default %s, got %s", defaultArtifactDir, got)
	}
}

// walkUpFor is the mechanism behind that anchoring: it must find a marker in
// an ancestor directory and report nothing when there is none.
func TestWalkUpFor(t *testing.T) {
	t.Parallel()

	t.Run("finds a marker in an ancestor", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "marker"), nil, 0o600); err != nil {
			t.Fatalf("setup: %v", err)
		}
		nested := filepath.Join(root, "a", "b")
		if err := os.MkdirAll(nested, 0o750); err != nil {
			t.Fatalf("setup: %v", err)
		}
		if got := walkUpFor(nested, "marker"); got != root {
			t.Fatalf("expected %s, got %s", root, got)
		}
	})

	t.Run("reports nothing when the marker is absent", func(t *testing.T) {
		t.Parallel()
		if got := walkUpFor(t.TempDir(), "definitely-not-present-marker"); got != "" {
			t.Fatalf("expected no root, got %s", got)
		}
	})
}

// A working directory that has been removed out from under the process leaves
// nothing to walk up from. The resolver must fall back rather than propagate
// the error, because artifact writing is a best-effort side channel.
//
//nolint:paralleltest // t.Chdir cannot be used from a parallel test
func TestFindModuleRootWithNoWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	// The nested dir is what gets removed; t.TempDir's own cleanup then has
	// a still-existing parent to work with.
	nested := filepath.Join(dir, "gone")
	if err := os.Mkdir(nested, 0o750); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Chdir(nested)
	if err := os.Remove(nested); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if got := findModuleRoot(); got != "" {
		t.Fatalf("a vanished working directory anchors nothing, got %s", got)
	}
}
