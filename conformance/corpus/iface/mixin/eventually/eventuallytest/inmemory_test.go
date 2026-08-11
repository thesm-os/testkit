// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package eventuallytest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/eventually"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/eventually/eventuallytest"
)

// eventually is the model tier's — AUTO-EVENTUAL-CONVERGENCE states it — so the
// suite generates the signature family alone.
//
// eidos held it out of the relational set deliberately: "observable eventually"
// raises *observable by what*, the same question `sideeffect` raises, and
// whether it joins that vocabulary is an open call. This fixture answers it
// structurally instead — Settle is the seam, so convergence is driven rather
// than waited for, and no clock is involved.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	eventuallytest.AssertMixedContract(t,
		eventuallytest.MixedSubject("in-memory", func() eventually.Mixed {
			return eventuallytest.NewInMemory()
		}),
	)
}

// A publish is not immediately visible and is visible after settling, which is
// the whole of the mixin — and unobservable in one call, since a subject that
// applied writes immediately satisfies every generated check.
func TestPublishIsVisibleOnlyAfterSettling(t *testing.T) {
	t.Parallel()

	s := eventuallytest.NewInMemory()
	ctx := t.Context()

	testkit.NoError(t, s.Publish(ctx, "one"), "publishing succeeds")

	before, err := s.Items(ctx)
	testkit.NoError(t, err, "listing succeeds")
	testkit.Len(t, before, 0, "and the publish has not landed yet")

	testkit.NoError(t, s.Settle(ctx), "settling succeeds")

	after, err := s.Items(ctx)
	testkit.NoError(t, err, "listing again succeeds")
	testkit.Equal(t, after, []string{"one"}, "and the publish has converged")
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	eventuallytest.AssertMixedContract(t,
		eventuallytest.MixedSubject("in-memory", func() eventually.Mixed {
			return eventuallytest.NewInMemory()
		}),
		eventuallytest.MixedWithout("Publish/smoke"),
		eventuallytest.MixedWithoutDouble(),
	)
}
