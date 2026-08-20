// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package crdtmergetest_test

import (
	"context"
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/crdtmerge"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/crdtmerge/crdtmergetest"
)

// Two interfaces, two runs, one implementation answering to both — which
// is the arrangement rather than an accident: a merge needs a peer, and the
// peer is a contract of its own precisely so a merge cannot reach into it.
//
// crdtmerge is the model tier's — AUTO-CRDT-MERGE states it — so the suite
// generates the signature family alone. The assignment is right for a reason
// this fixture shows plainly: convergence is a statement about two merges in
// opposite orders, and there is no single call that makes it.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	crdtmergetest.RunMixed(t,
		crdtmergetest.MixedHarness[*crdtmergetest.InMemory]{Name: "in-memory", New: crdtmergetest.NewInMemory},
		crdtmergetest.MixedChecks{
			{
				Method: "Merge",
				Name:   "folds-a-peer-in",
				Claim:  "Merge folds a peer in through its own interface",
				Run: func(tb testing.TB, s crdtmerge.Mixed, fx crdtmergetest.MixedFixture) {
					tb.Helper()
					// The derived peer is nil — an interface parameter admits
					// no literal — so a check wanting a real one builds it.
					// That is what the row table exists for.
					testkit.NoError(tb, s.Add(tb.Context(), fx.Item()), "the subject has an item")

					other := crdtmergetest.NewInMemory()
					testkit.NoError(tb, other.Add(tb.Context(), "theirs"), "the peer has one too")
					testkit.NoError(tb, s.Merge(tb.Context(), other), "merging succeeds")

					got, err := s.Items(tb.Context())
					testkit.NoError(tb, err, "listing succeeds")
					testkit.Assert(tb, got).Contains("theirs", "the peer's item arrived")
					testkit.Assert(tb, got).Contains(fx.Item(),
						"and a merge that discarded what was there would be a merge in name only")
				},
			},
			{
				Method: "Merge",
				Name:   "tolerates-a-missing-peer",
				Claim:  "Merge tolerates a peer that is not there",
				Run: func(tb testing.TB, s crdtmerge.Mixed, fx crdtmergetest.MixedFixture) {
					tb.Helper()
					// A nil peer reaches production through a replica that
					// failed to dial, and merging with nothing is a no-op
					// rather than a panic.
					testkit.NoError(tb, s.Merge(tb.Context(), nil),
						"merging with no peer changes nothing and reports nothing")
				},
			},
			{
				Method: "Merge",
				Name:   "reports-an-unreadable-peer",
				Claim:  "Merge reports a peer that cannot be read",
				Run: func(tb testing.TB, s crdtmerge.Mixed, fx crdtmergetest.MixedFixture) {
					tb.Helper()
					// Convergence is a claim about two replicas that both
					// answered. A merge that swallowed an unreachable peer
					// would report agreement with something it never read.
					testkit.ErrorIs(tb, s.Merge(tb.Context(), failingReplica{}), errPeerUnreadable,
						"the peer's failure is reported rather than merged over")
				},
			},
		},
	)
}

// The peer is a contract in its own right, and one implementation answers to
// both — which is what lets a merge read through the interface rather than
// reaching into a type it happens to know.
func TestReplicaContract(t *testing.T) {
	t.Parallel()

	crdtmergetest.RunReplica(t,
		crdtmergetest.ReplicaHarness[*crdtmergetest.InMemory]{Name: "in-memory", New: crdtmergetest.NewInMemory},
	)
}

// errPeerUnreadable is what failingReplica reports.
var errPeerUnreadable = errors.New("crdtmergetest_test: peer unreadable")

// failingReplica is a peer whose contents cannot be read.
type failingReplica struct{}

func (failingReplica) Items(context.Context) ([]string, error) { return nil, errPeerUnreadable }

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	crdtmergetest.RunMixed(t,
		crdtmergetest.MixedHarness[*crdtmergetest.InMemory]{Name: "in-memory", New: crdtmergetest.NewInMemory},
		crdtmergetest.MixedSuite.Without(crdtmergetest.MixedSuite.Checks.Add.Smoke()),
	)
}
