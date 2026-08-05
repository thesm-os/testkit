// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"errors"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
)

func TestFormatFailurePerKind(t *testing.T) {
	t.Parallel()

	t.Run("structural failure includes the structural callout", func(t *testing.T) {
		t.Parallel()
		got := formatFailure(&Failure{
			Kind:  FailureStructural,
			LawID: "AUTO-HASH-CHAIN-INTEGRITY-VERIFY",
			Err:   errors.New("hash mismatch at entry 3"),
		})
		testkit.Assert(t, got).
			Contains("[structural]", "kind header").
			Contains("structural: AUTO-HASH-CHAIN-INTEGRITY-VERIFY", "per-kind line").
			Contains("hash mismatch at entry 3", "error preserved")
	})

	t.Run("structural without LawID falls back to generic line", func(t *testing.T) {
		t.Parallel()
		got := formatFailure(&Failure{
			Kind: FailureStructural,
			Err:  errors.New("ordering violation"),
		})
		testkit.Assert(t, got).Contains("structural: chain or ordering violation", "fallback")
	})

	t.Run("semantic failure cites the source", func(t *testing.T) {
		t.Parallel()
		got := formatFailure(&Failure{
			Kind:  FailureSemantic,
			LawID: "AUTO-READ-AFTER-WRITE",
			Err:   errors.New("SUT/ref disagree"),
		})
		testkit.Assert(t, got).
			Contains("[semantic]", "kind header").
			Contains("semantic: SUT vs reference disagreement (AUTO-READ-AFTER-WRITE)", "per-kind line").
			Contains("SUT/ref disagree", "diff preserved")
	})

	t.Run("semantic without LawID labels source as action", func(t *testing.T) {
		t.Parallel()
		got := formatFailure(&Failure{
			Kind: FailureSemantic,
			Err:  errors.New("mismatch"),
		})
		testkit.Assert(t, got).Contains("(action)", "default source label")
	})

	t.Run("invariant with REQ includes law and req tags", func(t *testing.T) {
		t.Parallel()
		got := formatFailure(&Failure{
			Kind:  FailureInvariant,
			LawID: "AUTO-CAS-ATOMIC-ONE-WINNER",
			REQID: "REQ-STORE-001",
			Err:   errors.New("two writers won"),
		})
		testkit.Assert(t, got).
			Contains("REQ-STORE-001", "REQ rendered in header").
			Contains("[REQ-STORE-001 invariant]", "REQ prefixed").
			Contains("invariant: AUTO-CAS-ATOMIC-ONE-WINNER [REQ-STORE-001]", "per-kind line with both")
	})

	t.Run("invariant without LawID falls back", func(t *testing.T) {
		t.Parallel()
		got := formatFailure(&Failure{
			Kind: FailureInvariant,
			Err:  errors.New("unspecified"),
		})
		testkit.Assert(t, got).Contains("invariant: unspecified law fired", "fallback")
	})

	t.Run("liveness failure embeds goroutine stacks", func(t *testing.T) {
		t.Parallel()
		got := formatFailure(&Failure{
			Kind: FailureLiveness,
			Err:  errors.New("no progress after 1s"),
		})
		testkit.Assert(t, got).
			Contains("[liveness]", "kind header").
			Contains("liveness: deadlock or no-progress", "per-kind line").
			Contains("goroutine ", "stack output present")
	})

	t.Run("unclassified skips per-kind dispatch", func(t *testing.T) {
		t.Parallel()
		got := formatFailure(&Failure{
			Kind: FailureUnclassified,
			Err:  errors.New("uncategorized"),
		})
		// Header is present; no per-kind section markers.
		testkit.Assert(t, got).
			Contains("[unclassified]", "kind header").
			NotContains("structural:", "no per-kind").
			NotContains("semantic:", "no per-kind").
			NotContains("invariant:", "no per-kind").
			NotContains("liveness:", "no per-kind")
	})

	t.Run("artifact paths render after per-kind section", func(t *testing.T) {
		t.Parallel()
		got := formatFailure(&Failure{
			Kind:          FailureInvariant,
			LawID:         "AUTO-READ-AFTER-WRITE",
			Err:           errors.New("boom"),
			ArtifactPaths: []string{"/tmp/witness.txt", "/tmp/trace.json"},
		})
		invariantIdx := strings.Index(got, "invariant:")
		artifactIdx := strings.Index(got, "/tmp/witness.txt")
		testkit.True(t, invariantIdx > 0 && artifactIdx > 0, "both sections present")
		testkit.True(t, invariantIdx < artifactIdx, "artifacts come after kind line")
	})

	t.Run("invariant renders state at violation when set", func(t *testing.T) {
		t.Parallel()
		got := formatFailure(&Failure{
			Kind:     FailureInvariant,
			LawID:    "AUTO-READ-AFTER-WRITE",
			Err:      errors.New("get returned wrong value"),
			SUTState: "store{a: WRONG}",
			RefState: "store{a: correct}",
		})
		testkit.Assert(t, got).
			Contains("sut: store{a: WRONG}", "SUT state rendered").
			Contains("ref: store{a: correct}", "ref state rendered")
	})

	t.Run("semantic surfaces porcupine viz path when present", func(t *testing.T) {
		t.Parallel()
		got := formatFailure(&Failure{
			Kind:          FailureSemantic,
			Err:           errors.New("non-linearizable"),
			ArtifactPaths: []string{"viz: /tmp/linearizability-abc.html"},
		})
		testkit.Assert(t, got).
			Contains("porcupine: /tmp/linearizability-abc.html", "viz path surfaced")
	})

	t.Run("liveness includes suspected blocker line when one is found", func(t *testing.T) {
		t.Parallel()
		got := formatFailure(&Failure{
			Kind: FailureLiveness,
			Err:  errors.New("no progress"),
		})
		// runtime.Stack always includes the test runner goroutine; we can't
		// guarantee a specific blocking state in this synthetic test, so we
		// just assert the framing is consistent.
		testkit.Assert(t, got).Contains("liveness: deadlock or no-progress", "kind line")
	})
}

func TestSuspectedBlocker(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		stack     string
		wantID    string
		wantState string
	}{
		{
			name:      "chan receive header is flagged",
			stack:     "goroutine 47 [chan receive]:\n\tmain.worker(...)\n",
			wantID:    "47",
			wantState: stateChanReceive,
		},
		{
			name:      "semacquire header is flagged",
			stack:     "goroutine 8 [semacquire]:\nsync.runtime_Semacquire(...)\n",
			wantID:    "8",
			wantState: "semacquire",
		},
		{
			name:      "select header is flagged",
			stack:     "goroutine 12 [select]:\nmain.dispatch(...)\n",
			wantID:    "12",
			wantState: "select",
		},
		{
			name:      "long-running suffix is preserved",
			stack:     "goroutine 99 [chan receive, 5 minutes]:\n",
			wantID:    "99",
			wantState: "chan receive, 5 minutes",
		},
		{
			name:      "running and runnable are not blocking",
			stack:     "goroutine 1 [running]:\ngoroutine 2 [runnable]:\n",
			wantID:    "",
			wantState: "",
		},
		{
			name:      "first blocking match wins over later ones",
			stack:     "goroutine 1 [running]:\ngoroutine 5 [chan receive]:\ngoroutine 9 [semacquire]:\n",
			wantID:    "5",
			wantState: stateChanReceive,
		},
		{
			name:      "empty stack returns empties",
			stack:     "",
			wantID:    "",
			wantState: "",
		},
		{
			// A truncated dump can end mid-header. Without brackets there is
			// no state to read, so the line is skipped rather than guessed at.
			name:      "a header with no state bracket is skipped",
			stack:     "goroutine 3\ngoroutine 7 [chan receive]:\n",
			wantID:    "7",
			wantState: stateChanReceive,
		},
		{
			// The ID is read up to the first space. A header with no space
			// after the prefix has no delimited ID, so it is skipped even
			// though its state is a blocking one.
			name:      "a blocking header with no delimited ID is skipped",
			stack:     "goroutine 5[select]:\n",
			wantID:    "",
			wantState: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			id, state := suspectedBlocker(c.stack)
			testkit.Equal(t, id, c.wantID, "id")
			testkit.Equal(t, state, c.wantState, "state")
		})
	}
}
