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

// Two interfaces, two harnesses, one implementation answering to both — which
// is the arrangement rather than an accident: a merge needs a peer, and the
// peer is a contract of its own precisely so a merge cannot reach into it.
//
// crdtmerge is the model tier's — AUTO-CRDT-MERGE states it — so the suite
// generates the signature family alone. The assignment is right for a reason
// this fixture shows plainly: convergence is a statement about two merges in
// opposite orders, and there is no single call that makes it.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	crdtmergetest.AssertMixedContract(t,
		crdtmergetest.MixedModel(),
		crdtmergetest.MixedSubject("in-memory", func() crdtmerge.Mixed {
			return crdtmergetest.NewInMemory()
		}),
		crdtmergetest.MixedOnMerge("folds a peer in through its own interface", func(
			tb testing.TB, subject crdtmerge.Mixed, peer crdtmerge.Replica,
		) {
			tb.Helper()
			// The derived peer is nil — an interface parameter admits no
			// literal — so a check wanting a real one builds it. That is what
			// the extension point exists for.
			other := crdtmergetest.NewInMemory()
			testkit.NoError(tb, other.Add(tb.Context(), "theirs"), "the peer has an item")
			testkit.NoError(tb, subject.Merge(tb.Context(), other), "merging succeeds")

			// Present rather than sole: Add is classified writer, so the
			// harness has already seeded this subject through it, and a merge
			// that discarded what was there would be a merge in name only.
			got, err := subject.Items(tb.Context())
			testkit.NoError(tb, err, "listing succeeds")
			testkit.Assert(tb, got).Contains("theirs", "the peer's item arrived")
		}),
		crdtmergetest.MixedOnMerge("tolerates a peer that is not there", func(
			tb testing.TB, subject crdtmerge.Mixed, peer crdtmerge.Replica,
		) {
			tb.Helper()
			// A nil peer reaches production through a replica that failed to
			// dial, and merging with nothing is a no-op rather than a panic.
			testkit.NoError(tb, subject.Merge(tb.Context(), nil),
				"merging with no peer changes nothing and reports nothing")
		}),
		crdtmergetest.MixedOnMerge("reports a peer that cannot be read", func(
			tb testing.TB, subject crdtmerge.Mixed, peer crdtmerge.Replica,
		) {
			tb.Helper()
			// Convergence is a claim about two replicas that both answered. A
			// merge that swallowed an unreachable peer would report agreement
			// with something it never read.
			testkit.ErrorIs(tb, subject.Merge(tb.Context(), failingReplica{}), errPeerUnreadable,
				"the peer's failure is reported rather than merged over")
		}),
	)
}

// The peer is a contract in its own right, and one implementation answers to
// both — which is what lets a merge read through the interface rather than
// reaching into a type it happens to know.
func TestReplicaContract(t *testing.T) {
	t.Parallel()

	crdtmergetest.AssertReplicaContract(t,
		crdtmergetest.ReplicaModel(),
		crdtmergetest.ReplicaSubject("in-memory", func() crdtmerge.Replica {
			return crdtmergetest.NewInMemory()
		}),
	)
}

// errPeerUnreadable is what failingReplica reports.
var errPeerUnreadable = errors.New("crdtmergetest_test: peer unreadable")

// failingReplica is a peer whose contents cannot be read.
type failingReplica struct{}

func (failingReplica) Items(context.Context) ([]string, error) { return nil, errPeerUnreadable }

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	crdtmergetest.AssertMixedContract(t,
		crdtmergetest.MixedSubject("in-memory", func() crdtmerge.Mixed {
			return crdtmergetest.NewInMemory()
		}),
		crdtmergetest.MixedWithout("Add/smoke"),
		crdtmergetest.MixedWithoutDouble(),
	)
}
