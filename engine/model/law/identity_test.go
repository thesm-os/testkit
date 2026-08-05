// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model/law"
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
		// reader.go
		{
			name: "Cacheable", id: "AUTO-CACHEABLE", reqID: "",
			law: law.Cacheable[any, string, int]{},
		},
		{
			name: "DefaultOnError", id: "AUTO-DEFAULT-ON-ERROR", reqID: "",
			law: law.DefaultOnError[any, string, int]{},
		},
		{
			name: "PointInTime", id: "AUTO-POINT-IN-TIME", reqID: "",
			law: law.PointInTime[any, string, int]{},
		},
		{
			name: "Sticky", id: "AUTO-STICKY", reqID: "",
			law: &law.Sticky[any, string, int]{},
		},
		{
			name: "MonotonicNonDecreasing", id: "AUTO-MONOTONIC-NON-DECREASING", reqID: "",
			law: &law.MonotonicNonDecreasing[any, int]{},
		},
		// writer.go
		{
			name: "IdempotentWrite", id: "AUTO-IDEMPOTENT-WRITE", reqID: "",
			law: law.IdempotentWrite[any, string, int]{},
		},
		{
			name: "CommutativeWrite", id: "AUTO-COMMUTATIVE-WRITE", reqID: "",
			law: law.CommutativeWrite[any, string, int]{},
		},
		{
			name: "AtomicWrite", id: "AUTO-ATOMIC-WRITE", reqID: "",
			law: law.AtomicWrite[any, string, int]{},
		},
		{
			name: "ValidTransition", id: "AUTO-VALID-TRANSITION", reqID: "",
			law: law.ValidTransition[any, string, int]{},
		},
		// aggregator.go
		{
			name: "AggregatorBounded", id: "AUTO-AGGREGATOR-BOUNDED", reqID: "",
			law: law.AggregatorBounded[any, int]{},
		},
		{
			name: "Associative", id: "AUTO-ASSOCIATIVE", reqID: "",
			law: law.Associative[any, int, int]{},
		},
		{
			name: "Conservative", id: "AUTO-CONSERVATIVE", reqID: "",
			law: law.Conservative[any, int]{},
		},
		// stream.go (new entries appended)
		{
			name: "StreamCompletion", id: "AUTO-STREAM-COMPLETION", reqID: "",
			law: law.StreamCompletion[any, int]{},
		},
		{
			name: "StreamNoDuplicates", id: "AUTO-STREAM-NO-DUPLICATES", reqID: "",
			law: law.StreamNoDuplicates[any, int, int]{},
		},
		{
			name: "StreamStableOrder", id: "AUTO-STREAM-STABLE-ORDER", reqID: "",
			law: law.StreamStableOrder[any, int]{},
		},
		{
			name: "StreamPermutation", id: "AUTO-STREAM-PERMUTATION", reqID: "",
			law: law.StreamPermutation[any, int, int]{},
		},
		{
			name: "StreamOverMatch", id: "AUTO-STREAM-OVER-MATCH", reqID: "",
			law: law.StreamOverMatch[any, int, int]{},
		},
		// stateless.go
		{
			name: "Roundtrip", id: "AUTO-ROUNDTRIP", reqID: "",
			law: law.Roundtrip[any, int]{},
		},
		{
			name: "LossyRoundtrip", id: "AUTO-LOSSY-ROUNDTRIP", reqID: "",
			law: law.LossyRoundtrip[any, int]{},
		},
		{
			name: "TotalOver", id: "AUTO-TOTAL-OVER", reqID: "",
			law: law.TotalOver[any, int, int]{},
		},
		// lifecycle.go
		{
			name: "IdempotentLifecycle", id: "AUTO-IDEMPOTENT-LIFECYCLE", reqID: "",
			law: law.IdempotentLifecycle[any, int]{},
		},
		{
			name: "LeakFree", id: "AUTO-LEAK-FREE", reqID: "",
			law: law.LeakFree[any]{},
		},
		{
			name: "PoisonNilOnFresh", id: "AUTO-POISON-NIL-ON-FRESH", reqID: "",
			law: law.PoisonNilOnFresh[any]{},
		},
		{
			name: "PoisonIdempotentRead", id: "AUTO-POISON-IDEMPOTENT-READ", reqID: "",
			law: law.PoisonIdempotentRead[any]{},
		},
		// contract.go
		{
			name: "PersisterRetrievable", id: "AUTO-PERSISTER-RETRIEVABLE", reqID: "",
			law: law.PersisterRetrievable[any, int, int]{},
		},
		{
			name: "UpdaterReplaces", id: "AUTO-UPDATER-REPLACES", reqID: "",
			law: law.UpdaterReplaces[any, int, int]{},
		},
		{
			name: "UpserterIdempotent", id: "AUTO-UPSERTER-IDEMPOTENT", reqID: "",
			law: law.UpserterIdempotent[any, int, int]{},
		},
		{
			name: "CASAtomicOneWinner", id: "AUTO-CAS-ATOMIC-ONE-WINNER", reqID: "",
			law: law.CASAtomicOneWinner[any, int]{},
		},
		{
			name: "AppenderMonotonicOffsets", id: "AUTO-APPENDER-MONOTONIC-OFFSETS", reqID: "",
			law: &law.AppenderMonotonicOffsets[any, int, int64]{},
		},
		{
			name: "SingleflightCoalesces", id: "AUTO-SINGLEFLIGHT-COALESCES", reqID: "",
			law: law.SingleflightCoalesces[any, int, int]{},
		},
		{
			name: "TransactionRollbackOnError", id: "AUTO-TRANSACTION-ROLLBACK", reqID: "",
			law: law.TransactionRollbackOnError[any, int, int]{},
		},
		{
			name: "LeaseDoubleAcquireBlocks", id: "AUTO-LEASE-DOUBLE-ACQUIRE-BLOCKS", reqID: "",
			law: law.LeaseDoubleAcquireBlocks[any, int]{},
		},
		// composite.go
		{
			name: "PoolBalancedGetPut", id: "AUTO-POOL-BALANCED", reqID: "",
			law: law.PoolBalancedGetPut[any]{},
		},
		{
			name: "PoolLeakFree", id: "AUTO-POOL-LEAK-FREE", reqID: "",
			law: law.PoolLeakFree[any]{},
		},
		{
			name: "CursorCloseIdempotent", id: "AUTO-CURSOR-CLOSE-IDEMPOTENT", reqID: "",
			law: law.CursorCloseIdempotent[any]{},
		},
		{
			name: "CursorNextAfterCloseSentinel", id: "AUTO-CURSOR-NEXT-AFTER-CLOSE", reqID: "",
			law: law.CursorNextAfterCloseSentinel[any, int]{},
		},
		{
			name: "TwoPhaseNoRollbackAfterCommit", id: "AUTO-TWO-PHASE-ROLLBACK-AFTER-COMMIT", reqID: "",
			law: law.TwoPhaseNoRollbackAfterCommit[any, int]{},
		},
		{
			name: "SagaFullCompensation", id: "AUTO-SAGA-FULL-COMPENSATION", reqID: "",
			law: law.SagaFullCompensation[any, int]{},
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
