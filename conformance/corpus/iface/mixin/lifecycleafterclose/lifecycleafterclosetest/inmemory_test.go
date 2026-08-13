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
		lifecycleafterclosetest.MixedModel(),
		lifecycleafterclosetest.MixedSubject("in-memory", func() lifecycleafterclose.Mixed {
			return lifecycleafterclosetest.NewInMemory()
		}),
		lifecycleafterclosetest.MixedOnWork("refuses work after the subject closed", func(
			tb testing.TB, subject lifecycleafterclose.Mixed,
		) {
			tb.Helper()
			// The mixin's own claim, and one no single call can make: what
			// changes is the state between the two.
			testkit.NoError(tb, subject.Close(tb.Context()), "the subject closes")
			testkit.Error(tb, subject.Work(tb.Context()),
				"and work after it is refused rather than quietly done")
		}),
	)
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

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestMixedSaturation(t *testing.T) {
	t.Parallel()
	lifecycleafterclosetest.MixedModelSaturation(t, func() lifecycleafterclose.Mixed {
		return lifecycleafterclosetest.NewInMemory()
	})
}
