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
			LawID: "AUTO-HASH-CHAIN-INTEGRITY",
			Err:   errors.New("hash mismatch at entry 3"),
		})
		testkit.Assert(t, got).
			Contains("[structural]", "kind header").
			Contains("structural: AUTO-HASH-CHAIN-INTEGRITY", "per-kind line").
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
}
