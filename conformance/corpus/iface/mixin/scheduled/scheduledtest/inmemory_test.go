// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package scheduledtest_test

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/scheduled"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/scheduled/scheduledtest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	scheduledtest.AssertMixedContract(t,
		scheduledtest.MixedModel(scheduledtest.MixedModelClocked(
			func(clk *clock.TestClock) scheduled.Mixed { return scheduledtest.NewInMemoryOn(clk) },
		)),
		scheduledtest.MixedSubject("in-memory", func() scheduled.Mixed {
			return scheduledtest.NewInMemory()
		}),
		scheduledtest.MixedOnFired("counts nothing before its instant arrives", func(
			tb testing.TB, subject scheduled.Mixed,
		) {
			tb.Helper()
			// A task registered for the future has not run on a clock nobody
			// advanced. Asserting the zero is what stops this fixture from
			// passing against a scheduler that fires everything immediately.
			testkit.NoError(tb, subject.At(tb.Context(), time.Hour), "the task registers")

			got, err := subject.Fired(tb.Context())
			testkit.NoError(tb, err, "the count is readable")
			testkit.Equal(tb, got, 0, "an hour has not passed on this clock")

			// A task due now is already due: its instant is not after the
			// clock's reading. That is the other side of the same comparison,
			// and without it a scheduler that never fires would pass.
			testkit.NoError(tb, subject.At(tb.Context(), 0), "a task due now registers")

			got, err = subject.Fired(tb.Context())
			testkit.NoError(tb, err, "the count is readable")
			testkit.Equal(tb, got, 1, "and counts the one whose instant has arrived")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	scheduledtest.AssertMixedContract(t,
		scheduledtest.MixedSubject("in-memory", func() scheduled.Mixed {
			return scheduledtest.NewInMemory()
		}),
		scheduledtest.MixedWithoutDouble(),
	)
}

// The saturation prover: every bound law must be able to fail as itself;
// the clocked laws skip themselves — their factory is the clock's.
func TestMixedSaturation(t *testing.T) {
	t.Parallel()
	scheduledtest.MixedModelSaturation(t, func() scheduled.Mixed {
		return scheduledtest.NewInMemory()
	})
}
