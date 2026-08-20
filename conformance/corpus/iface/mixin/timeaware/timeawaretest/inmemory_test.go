// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package timeawaretest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/timeaware"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/timeaware/timeawaretest"
)

// The generated contract, run against the in-memory subject.
//
// timeaware has no suite-side rule yet — the header says so — so what the
// classification means for this pair is the row's.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	timeawaretest.RunMixed(t,
		timeawaretest.MixedHarness[*timeawaretest.InMemory]{Name: "in-memory", New: timeawaretest.NewInMemory},
		timeawaretest.MixedChecks{
			{
				Method: "AgeOf",
				Name:   "answers-from-the-clock",
				Claim:  "AgeOf answers from the clock rather than the wall",
				Run: func(tb testing.TB, s timeaware.Mixed, fx timeawaretest.MixedFixture) {
					tb.Helper()
					testkit.NoError(tb, s.Touch(tb.Context(), fx.Key()), "the key is recorded")

					// On a clock nothing advanced the age is zero — and reading
					// it twice gives the same answer, which a wall-clock subject
					// could not promise.
					first, err := s.AgeOf(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "the touched key has an age")
					testkit.Equal(tb, first, int64(0), "no time has passed on this clock")

					again, err := s.AgeOf(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "and it is still readable")
					testkit.Equal(tb, again, first, "a clock nobody advanced does not move")
				},
			},
		},
	)
}
