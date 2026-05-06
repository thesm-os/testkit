// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"errors"
	"strings"
	"testing"

	"go.thesmos.sh/testkit/model/trace"
)

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
				{ClientID: -1, OpName: "Put", Inputs: []any{"item-a"}, Output: nil},
				{ClientID: -1, OpName: "Put", Inputs: []any{"item-b"}, Output: nil},
				{ClientID: -1, OpName: "Get", Inputs: []any{"x"}, Output: "old"},
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
				{ClientID: -1, OpName: "Append", Inputs: []any{"entry-1"}, Output: nil},
				{ClientID: -1, OpName: "Verify", Inputs: nil, Output: nil, Err: errors.New("chain integrity")},
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
				{ClientID: -1, OpName: "Put", Inputs: []any{"v"}, Output: nil},
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
		if len(got) > 50 {
			// Should be truncated to ~32 chars + quotes + ...
		}
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
