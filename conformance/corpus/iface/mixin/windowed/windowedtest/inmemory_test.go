// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package windowedtest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/windowed"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/windowed/windowedtest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	windowedtest.AssertMixedContract(t,
		windowedtest.MixedModel(windowedtest.MixedModelClocked(
			func(clk *clock.TestClock) windowed.Mixed { return windowedtest.NewInMemoryOn(clk) },
		)),
		windowedtest.MixedSubject("in-memory", func() windowed.Mixed {
			return windowedtest.NewInMemory()
		}),
		windowedtest.MixedOnCountIn("counts inside the window and not outside it", func(
			tb testing.TB, subject windowed.Mixed, key string,
		) {
			tb.Helper()
			// The suite seeds through Record, so one occurrence is already
			// inside the window on a clock nothing advanced. A key nothing
			// recorded counts zero rather than erroring — an absent key has
			// no occurrences, which is an answer and not a failure.
			got, err := subject.CountIn(tb.Context(), key)
			testkit.NoError(tb, err, "the count is readable")
			testkit.Equal(tb, got, 1, "the seeded occurrence is inside the window")

			absent, err := subject.CountIn(tb.Context(), key+"-absent")
			testkit.NoError(tb, err, "an unrecorded key is not an error")
			testkit.Equal(tb, absent, 0, "it simply has no occurrences")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	windowedtest.AssertMixedContract(t,
		windowedtest.MixedSubject("in-memory", func() windowed.Mixed {
			return windowedtest.NewInMemory()
		}),
		windowedtest.MixedWithoutDouble(),
	)
}
