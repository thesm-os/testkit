// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package lifecycletest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/lifecycle"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/lifecycle/lifecycletest"
)

// Close carries the idempotent mixin, so Close/idempotent is derived and the
// row below is the same claim written by hand — kept as the worked example of
// a row standing beside the generated check it duplicates.
func TestLifecycleContract(t *testing.T) {
	t.Parallel()

	lifecycletest.RunLifecycle(t,
		lifecycletest.LifecycleHarness[*lifecycletest.InMemory]{Name: "in-memory", New: lifecycletest.NewInMemory},
		lifecycletest.LifecycleChecks{
			{
				Method: "Close",
				Name:   "second-close-succeeds",
				Claim:  "Close is idempotent",
				Run: func(tb testing.TB, s lifecycle.Lifecycle, fx lifecycletest.LifecycleFixture) {
					tb.Helper()
					testkit.NoError(tb, s.Close(tb.Context()), "the first close succeeds")
					testkit.NoError(tb, s.Close(tb.Context()), "and so does the second")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestLifecycleContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	lifecycletest.RunLifecycle(t,
		lifecycletest.LifecycleHarness[*lifecycletest.InMemory]{Name: "in-memory", New: lifecycletest.NewInMemory},
		lifecycletest.LifecycleSuite.Without(lifecycletest.LifecycleSuite.Checks.Close.Smoke()),
	)
}
