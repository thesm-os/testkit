// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package windowedtest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/windowed"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/windowed/windowedtest"
)

// The generated contract, run against the in-memory subject.
//
// What falls OUT of the window needs a clock the run advances, so it is the
// model tier's. What is inside one is the row's.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	windowedtest.RunMixed(t,
		windowedtest.MixedHarness[*windowedtest.InMemory]{Name: "in-memory", New: windowedtest.NewInMemory},
		windowedtest.MixedChecks{
			{
				Method: "CountIn",
				Name:   "counts-inside-the-window",
				Claim:  "CountIn counts inside the window and not outside it",
				Run: func(tb testing.TB, s windowed.Mixed, fx windowedtest.MixedFixture) {
					tb.Helper()
					// One occurrence, recorded on a clock nothing advanced, so
					// it is inside the window by construction.
					testkit.NoError(tb, s.Record(tb.Context(), fx.Key()), "an occurrence is recorded")

					got, err := s.CountIn(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "the count is readable")
					testkit.Equal(tb, got, 1, "the recorded occurrence is inside the window")

					// A key nothing recorded counts zero rather than erroring —
					// an absent key has no occurrences, which is an answer and
					// not a failure.
					absent, err := s.CountIn(tb.Context(), fx.KeyOther())
					testkit.NoError(tb, err, "an unrecorded key is not an error")
					testkit.Equal(tb, absent, 0, "it simply has no occurrences")
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

	windowedtest.RunMixed(t,
		windowedtest.MixedHarness[*windowedtest.InMemory]{Name: "in-memory", New: windowedtest.NewInMemory},
		windowedtest.MixedSuite.Without(windowedtest.MixedSuite.Checks.Record.Smoke()),
	)
}
