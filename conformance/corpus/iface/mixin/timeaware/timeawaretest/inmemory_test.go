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
func TestMixedContract(t *testing.T) {
	t.Parallel()

	timeawaretest.AssertMixedContract(t,
		timeawaretest.MixedModel(),
		timeawaretest.MixedSubject("in-memory", func() timeaware.Mixed {
			return timeawaretest.NewInMemory()
		}),
		timeawaretest.MixedOnAgeOf("answers from the clock rather than the wall", func(
			tb testing.TB, subject timeaware.Mixed, key string,
		) {
			tb.Helper()
			// The suite seeds through Touch, so the key is recorded. On a
			// clock nothing advanced, the age is zero — and reading it twice
			// gives the same answer, which a wall-clock subject could not
			// promise.
			first, err := subject.AgeOf(tb.Context(), key)
			testkit.NoError(tb, err, "the touched key has an age")
			testkit.Equal(tb, first, int64(0), "no time has passed on this clock")

			again, err := subject.AgeOf(tb.Context(), key)
			testkit.NoError(tb, err, "and it is still readable")
			testkit.Equal(tb, again, first, "a clock nobody advanced does not move")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	timeawaretest.AssertMixedContract(t,
		timeawaretest.MixedSubject("in-memory", func() timeaware.Mixed {
			return timeawaretest.NewInMemory()
		}),
		timeawaretest.MixedWithoutDouble(),
	)
}
