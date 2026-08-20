// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package scheduledtest_test

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/scheduled"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/scheduled/scheduledtest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	scheduledtest.RunMixed(t,
		scheduledtest.MixedHarness[*scheduledtest.InMemory]{Name: "in-memory", New: scheduledtest.NewInMemory},
		scheduledtest.MixedChecks{
			{
				Method: "Fired",
				Name:   "counts-nothing-before-its-instant",
				Claim:  "Fired counts nothing before its instant arrives",
				Run: func(tb testing.TB, s scheduled.Mixed, fx scheduledtest.MixedFixture) {
					tb.Helper()
					// A task registered for the future has not run on a clock
					// nobody advanced. Asserting the zero is what stops this
					// fixture from passing against a scheduler that fires
					// everything immediately.
					testkit.NoError(tb, s.At(tb.Context(), time.Hour), "the task registers")

					got, err := s.Fired(tb.Context())
					testkit.NoError(tb, err, "the count is readable")
					testkit.Equal(tb, got, 0, "an hour has not passed on this clock")

					// A task due now is already due: its instant is not after
					// the clock's reading. That is the other side of the same
					// comparison, and without it a scheduler that never fires
					// would pass.
					testkit.NoError(tb, s.At(tb.Context(), 0), "a task due now registers")

					got, err = s.Fired(tb.Context())
					testkit.NoError(tb, err, "the count is readable")
					testkit.Equal(tb, got, 1, "and counts the one whose instant has arrived")
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

	scheduledtest.RunMixed(t,
		scheduledtest.MixedHarness[*scheduledtest.InMemory]{Name: "in-memory", New: scheduledtest.NewInMemory},
		scheduledtest.MixedSuite.Without(scheduledtest.MixedSuite.Checks.Fired.Smoke()),
	)
}
