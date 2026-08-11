// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package lifecycleafterclosetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/lifecycleafterclose"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/lifecycleafterclose/lifecycleafterclosetest"
)

// lifecycleafterclose is the model tier's — AUTO-LIFECYCLE-AFTER-CLOSE states
// it — so the suite generates the signature family alone.
//
// The assignment is right for a reason the subject shows: the claim is about
// what a call does *after* another call, and every generated check runs against
// a fresh subject precisely so no check depends on what another left behind.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	lifecycleafterclosetest.AssertMixedContract(t,
		lifecycleafterclosetest.MixedSubject("in-memory", func() lifecycleafterclose.Mixed {
			return lifecycleafterclosetest.NewInMemory()
		}),
	)
}

// Every later operation reports the closed sentinel, and does no work while
// reporting it. A subject that returned the error and worked anyway satisfies
// any check reading only the error.
func TestWorkRefusesAfterClose(t *testing.T) {
	t.Parallel()

	s := lifecycleafterclosetest.NewInMemory()
	testkit.NoError(t, s.Work(t.Context()), "an open subject works")
	testkit.Equal(t, s.Works(), 1, "and the work landed")

	testkit.NoError(t, s.Close(t.Context()), "closing succeeds")
	testkit.ErrorIs(t, s.Work(t.Context()), lifecycleafterclosetest.ErrClosed,
		"and every later operation reports the closed sentinel")
	testkit.Equal(t, s.Works(), 1, "having done no further work")
}

// Close is idempotent, which a caller with a deferred close and an explicit one
// relies on.
func TestCloseIsIdempotent(t *testing.T) {
	t.Parallel()

	s := lifecycleafterclosetest.NewInMemory()
	testkit.NoError(t, s.Close(t.Context()), "the first close succeeds")
	testkit.NoError(t, s.Close(t.Context()), "and so does the second")
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	lifecycleafterclosetest.AssertMixedContract(t,
		lifecycleafterclosetest.MixedSubject("in-memory", func() lifecycleafterclose.Mixed {
			return lifecycleafterclosetest.NewInMemory()
		}),
		lifecycleafterclosetest.MixedWithout("Work/smoke"),
		lifecycleafterclosetest.MixedWithoutDouble(),
	)
}
