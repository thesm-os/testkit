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
	)
}

// The peer is a contract in its own right, and one implementation answers to
// both — which is what lets a merge read through the interface rather than
// reaching into a type it happens to know.
func TestReplicaContract(t *testing.T) {
	t.Parallel()

	crdtmergetest.AssertReplicaContract(t,
		crdtmergetest.ReplicaSubject("in-memory", func() crdtmerge.Replica {
			return crdtmergetest.NewInMemory()
		}),
	)
}

// Merging in either order arrives at the same set, which is the whole of the
// mixin and needs four replicas to state.
func TestMergeConverges(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	left, right := crdtmergetest.NewInMemory(), crdtmergetest.NewInMemory()
	testkit.NoError(t, left.Add(ctx, "a"), "left diverges")
	testkit.NoError(t, right.Add(ctx, "b"), "right diverges")

	leftFirst := crdtmergetest.NewInMemory()
	testkit.NoError(t, leftFirst.Merge(ctx, left), "merging left succeeds")
	testkit.NoError(t, leftFirst.Merge(ctx, right), "then right")

	rightFirst := crdtmergetest.NewInMemory()
	testkit.NoError(t, rightFirst.Merge(ctx, right), "merging right succeeds")
	testkit.NoError(t, rightFirst.Merge(ctx, left), "then left")

	a, err := leftFirst.Items(ctx)
	testkit.NoError(t, err, "listing one order succeeds")
	b, err := rightFirst.Items(ctx)
	testkit.NoError(t, err, "listing the other succeeds")
	testkit.Equal(t, a, b, "both orders arrive at the same set")
}

// Merging twice changes nothing, which is the idempotence half of the same
// property.
func TestMergeIsIdempotent(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	peer := crdtmergetest.NewInMemory()
	testkit.NoError(t, peer.Add(ctx, "x"), "the peer has an item")

	s := crdtmergetest.NewInMemory()
	testkit.NoError(t, s.Merge(ctx, peer), "the first merge succeeds")
	testkit.NoError(t, s.Merge(ctx, peer), "and so does the second")

	got, err := s.Items(ctx)
	testkit.NoError(t, err, "listing succeeds")
	testkit.Equal(t, got, []string{"x"}, "the second merge changed nothing")
}

// A nil peer is nothing to fold in rather than a crash. It reaches production
// through a caller that had no replica yet, and it is what the fixture's own
// derived value is — an interface parameter admits no literal.
func TestMergeToleratesANilPeer(t *testing.T) {
	t.Parallel()

	s := crdtmergetest.NewInMemory()
	testkit.NoError(t, s.Merge(t.Context(), nil), "merging nothing succeeds")

	got, err := s.Items(t.Context())
	testkit.NoError(t, err, "listing succeeds")
	testkit.Len(t, got, 0, "and nothing arrived")
}

// A peer that cannot be read fails the merge rather than half-applying it.
func TestMergeReportsAFailingPeer(t *testing.T) {
	t.Parallel()

	s := crdtmergetest.NewInMemory()
	testkit.ErrorIs(t, s.Merge(t.Context(), failingReplica{}), errPeerUnreadable,
		"a peer that will not list is a failed merge")
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
