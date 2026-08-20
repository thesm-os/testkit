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

	lifecycleafterclosetest.RunMixed(
		t,
		lifecycleafterclosetest.MixedHarness[*lifecycleafterclosetest.InMemory]{
			Name: "in-memory",
			New:  lifecycleafterclosetest.NewInMemory,
		},
		lifecycleafterclosetest.MixedChecks{
			{
				Method: "Work",
				Name:   "refused-after-close",
				Claim:  "Work refuses work after the subject closed",
				Run: func(tb testing.TB, s lifecycleafterclose.Mixed, fx lifecycleafterclosetest.MixedFixture) {
					tb.Helper()
					// The mixin's own claim, and one no single call can make:
					// what changes is the state between the two.
					testkit.NoError(tb, s.Close(tb.Context()), "the subject closes")
					testkit.Error(tb, s.Work(tb.Context()),
						"and work after it is refused rather than quietly done")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	lifecycleafterclosetest.RunMixed(
		t,
		lifecycleafterclosetest.MixedHarness[*lifecycleafterclosetest.InMemory]{
			Name: "in-memory",
			New:  lifecycleafterclosetest.NewInMemory,
		},
		lifecycleafterclosetest.MixedSuite.Without(lifecycleafterclosetest.MixedSuite.Checks.Work.Smoke()),
	)
}
