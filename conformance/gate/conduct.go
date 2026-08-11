// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

import "go.thesmos.sh/testkit/core/lawid"

// Conduct is how a law's Check treats the shared (sut, ref) pair the runner
// hands it — the contract [engine/model/law.Law] states, recorded per law so
// it can be enforced rather than remembered.
//
// The first four conducts keep the pair synchronized and may be bound by the
// generator. The last two do not, and a law carrying one must not gain an
// instantiation row until it is fixed or the runner grows an isolation
// mechanism — which is exactly what the census test holds.
type Conduct string

// The conduct vocabulary.
const (
	// ConductObservational reads only.
	ConductObservational Conduct = "observational"

	// ConductMirrored lands every accepted mutation on both sides.
	ConductMirrored Conduct = "mirrored"

	// ConductSelfCleaning mutates and restores within one Check — an acquire
	// released, a put deleted, a transaction rolled back.
	ConductSelfCleaning Conduct = "self-cleaning"

	// ConductIsolated builds its own subjects through a Factory field and
	// never touches the pair.
	ConductIsolated Conduct = "isolated"

	// ConductNeedsMirror mutates the subject without mirroring — sound alone,
	// unsound interleaved. The fix is mechanical (the mirror helper) and owed
	// before the law's instantiation row lands.
	ConductNeedsMirror Conduct = "needs-mirror"

	// ConductNeedsIsolation corrupts or kills the subject to make its
	// observation — tampering its state, closing it, poisoning it. No mirror
	// repairs that; the law needs a subject of its own, which the runner does
	// not yet offer.
	ConductNeedsIsolation Conduct = "needs-isolation"
)

// Sound reports whether the conduct keeps a shared pair synchronized.
func (c Conduct) Sound() bool {
	switch c {
	case ConductObservational, ConductMirrored, ConductSelfCleaning, ConductIsolated:
		return true
	case ConductNeedsMirror, ConductNeedsIsolation:
		return false
	}
	return false
}

// LawConduct classifies every law in the catalogue.
//
// The classification is by reading each Check — there is nothing mechanical
// that can see a mutation through a closure — which is why it lives here,
// where a test holds it total over the vocabulary and the binding column to
// its verdicts. A law added without a row fails the census by name.
//
//nolint:gochecknoglobals // a census table, read-only, test-facing.
var LawConduct = map[string]Conduct{
	lawid.AggregatorBounded:        ConductObservational,
	lawid.AppendOnlyGrows:          ConductObservational,
	lawid.AppendOnlyNoDrops:        ConductObservational,
	lawid.Cacheable:                ConductObservational,
	lawid.CausalOrdering:           ConductObservational,
	lawid.CountEqualsReference:     ConductObservational,
	lawid.DeadlineRespecting:       ConductObservational,
	lawid.DefaultOnError:           ConductObservational,
	lawid.DeleteReturnsNotFound:    ConductObservational,
	lawid.HashChainIntegrityErr:    ConductObservational,
	lawid.HashChainIntegrityVerify: ConductObservational,
	lawid.LifecycleRespectsContext: ConductObservational,
	lawid.LossyRoundtrip:           ConductObservational,
	lawid.MonotonicNonDecreasing:   ConductObservational,
	lawid.MonotonicReads:           ConductObservational,
	lawid.MonotonicWrites:          ConductObservational,
	lawid.PaginatorNoDuplicates:    ConductObservational,
	lawid.PaginatorResumable:       ConductObservational,
	lawid.PoisonIdempotentRead:     ConductObservational,
	lawid.PoolBalanced:             ConductObservational,
	lawid.PoolLeakFree:             ConductObservational,
	lawid.PredicateConsistent:      ConductObservational,
	lawid.PureDeterministic:        ConductObservational,
	lawid.ReadAfterWrite:           ConductObservational,
	lawid.ReadYourWrites:           ConductObservational,
	lawid.ReplayCausalOrdering:     ConductObservational,
	lawid.ReplayDeterministic:      ConductObservational,
	lawid.Roundtrip:                ConductObservational,
	lawid.SnapshotIsolationG0:      ConductObservational,
	lawid.SnapshotIsolationG1:      ConductObservational,
	lawid.SnapshotIsolationG2:      ConductObservational,
	lawid.Sticky:                   ConductObservational,
	lawid.StreamCompletion:         ConductObservational,
	lawid.StreamNoDuplicates:       ConductObservational,
	lawid.StreamOverMatch:          ConductObservational,
	lawid.StreamPermutation:        ConductObservational,
	lawid.StreamReentrant:          ConductObservational,
	lawid.StreamStableOrder:        ConductObservational,
	lawid.TotalOver:                ConductObservational,
	lawid.WritesFollowReads:        ConductObservational,
	lawid.XSSSafe:                  ConductObservational,

	lawid.AtomicWrite:     ConductMirrored,
	lawid.Conservative:    ConductMirrored,
	lawid.IdempotentWrite: ConductMirrored,
	lawid.InjectionSafe:   ConductMirrored,
	lawid.PointInTime:     ConductMirrored,
	lawid.ValidTransition: ConductMirrored,
	lawid.Windowed:        ConductMirrored,
	lawid.WriteObservable: ConductMirrored,

	lawid.LeakFree:                     ConductSelfCleaning,
	lawid.LeaseDoubleAcquireBlocks:     ConductSelfCleaning,
	lawid.LeaseReleasedOnCancel:        ConductSelfCleaning,
	lawid.StreamReflectsMutations:      ConductSelfCleaning,
	lawid.TransactionNoMidTxVisibility: ConductSelfCleaning,
	lawid.TransactionRollback:          ConductSelfCleaning,
	lawid.TwoPhaseMutex:                ConductSelfCleaning,
	lawid.TwoPhaseRollbackAfterCommit:  ConductSelfCleaning,

	lawid.Associative:         ConductIsolated,
	lawid.CRDTMerge:           ConductIsolated,
	lawid.CommutativeWrite:    ConductIsolated,
	lawid.EventualConvergence: ConductIsolated,
	lawid.PoisonNilOnFresh:    ConductIsolated,

	lawid.AppenderMonotonicOffsets:   ConductNeedsMirror,
	lawid.CASAtomicOneWinner:         ConductNeedsMirror,
	lawid.PersisterRetrievable:       ConductNeedsMirror,
	lawid.PublisherAtLeastOnce:       ConductNeedsMirror,
	lawid.PublisherAtMostOnce:        ConductNeedsMirror,
	lawid.PublisherDelivers:          ConductNeedsMirror,
	lawid.PublisherDelivery:          ConductNeedsMirror,
	lawid.PublisherExactlyOnce:       ConductNeedsMirror,
	lawid.SagaFullCompensation:       ConductNeedsMirror,
	lawid.ScheduledFiresAfterAdvance: ConductNeedsMirror,
	lawid.SingleflightCoalesces:      ConductNeedsMirror,
	lawid.TTLExpiry:                  ConductNeedsMirror,
	lawid.UpdaterReplaces:            ConductNeedsMirror,
	lawid.UpserterIdempotent:         ConductNeedsMirror,
	lawid.WatcherReturnsOnChange:     ConductNeedsMirror,

	lawid.CursorCloseIdempotent: ConductNeedsIsolation,
	lawid.CursorNextAfterClose:  ConductNeedsIsolation,
	lawid.IdempotentLifecycle:   ConductNeedsIsolation,
	lawid.LifecycleAfterClose:   ConductNeedsIsolation,
	lawid.PoisonConsistent:      ConductNeedsIsolation,
	lawid.TamperEvident:         ConductNeedsIsolation,
}
