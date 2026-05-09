// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/law"
)

// TestLawIdentities verifies ID() and REQID() on every shipped law
// type. These are trivial one-liners but contribute to coverage and
// catch accidental drift in the stable identifiers.
func TestLawIdentities(t *testing.T) {
	t.Parallel()

	type lawEntry struct {
		name  string
		id    string
		reqID string
		law   law.Law[any]
	}

	tests := []lawEntry{
		// law.go
		{
			name: "ReadAfterWrite", id: "AUTO-READ-AFTER-WRITE", reqID: "",
			law: law.ReadAfterWrite[any, string, any]{
				Read: func(_ *rapid.T, _ any, _ string) (any, error) { return nil, nil },
				Keys: rapid.Just("k"),
			},
		},
		{
			name: "DeleteReturnsNotFound", id: "AUTO-DELETE-RETURNS-NOT-FOUND", reqID: "",
			law: law.DeleteReturnsNotFound[any, string, any]{
				Read: func(_ *rapid.T, _ any, _ string) (any, error) { return nil, nil },
				Keys: rapid.Just("k"),
			},
		},
		{
			name: "CountEqualsReference", id: "AUTO-COUNT-EQUALS-REFERENCE", reqID: "",
			law: law.CountEqualsReference[any, int]{
				Count: func(_ *rapid.T, _ any) (int, error) { return 0, nil },
			},
		},
		// predicate.go
		{
			name: "PredicateConsistency", id: "AUTO-PREDICATE-CONSISTENT", reqID: "",
			law: law.PredicateConsistency[any]{
				Call: func(_ *rapid.T, _ any) bool { return true },
			},
		},
		// pure.go
		{
			name: "PureDeterminism", id: "AUTO-PURE-DETERMINISTIC", reqID: "",
			law: law.PureDeterminism[any, int]{
				Call: func(_ *rapid.T, _ any) int { return 42 },
			},
		},
		// stream.go
		{
			name: "StreamReentrancy", id: "AUTO-STREAM-REENTRANT", reqID: "",
			law: law.StreamReentrancy[any, int]{
				Collect: func(_ *rapid.T, _ any) ([]int, error) { return nil, nil },
			},
		},
		// chain.go
		{
			name: "AppendOnlyHistoryGrows", id: "AUTO-APPEND-ONLY-GROWS", reqID: "",
			law: &law.AppendOnlyHistoryGrows[any, string, int]{},
		},
		{
			name: "AppendOnlyNoDrops", id: "AUTO-APPEND-ONLY-NO-DROPS", reqID: "",
			law: law.AppendOnlyNoDrops[any, string, int]{},
		},
		{
			name: "HashChainIntegrityViaVerify", id: "AUTO-HASH-CHAIN-INTEGRITY", reqID: "",
			law: law.HashChainIntegrityViaVerify[any]{},
		},
		{
			name: "HashChainIntegrityViaErr", id: "AUTO-HASH-CHAIN-INTEGRITY", reqID: "",
			law: law.HashChainIntegrityViaErr[any]{},
		},
		{
			name: "ReplayDeterminism", id: "AUTO-REPLAY-DETERMINISTIC", reqID: "",
			law: law.ReplayDeterminism[any, string, int]{},
		},
		{
			name: "ReplayRespectsCausality", id: "AUTO-REPLAY-CAUSAL-ORDERING", reqID: "",
			law: law.ReplayRespectsCausality[any, string, int]{},
		},
		// trace.go
		{
			name: "AfterEvery", id: "TRACE-AFTER-EVERY-Put", reqID: "",
			law: &law.AfterEvery[any]{ActionName: "Put"},
		},
		{
			name: "EventuallyAfter", id: "TRACE-EVENTUALLY-Flush-AFTER-Write", reqID: "",
			law: &law.EventuallyAfter[any]{Trigger: "Write", Response: "Flush"},
		},
		{
			name: "Never", id: "TRACE-NEVER-Crash", reqID: "",
			law: &law.Never[any]{ActionName: "Crash"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/ID", func(t *testing.T) {
			t.Parallel()
			if got := tt.law.ID(); got != tt.id {
				t.Fatalf("ID: got %q, want %q", got, tt.id)
			}
		})
		t.Run(tt.name+"/REQID", func(t *testing.T) {
			t.Parallel()
			if got := tt.law.REQID(); got != tt.reqID {
				t.Fatalf("REQID: got %q, want %q", got, tt.reqID)
			}
		})
	}
}
