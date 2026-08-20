// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package totaltest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/total"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/total/totaltest"
)

// A total method has no miss, which is what `total` says and what the header
// now records: the reader shape's miss check was refused here rather than
// emitted against a claim the classification denies.
//
// The domain's edge is the row's: the empty string.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	totaltest.RunMixed(t,
		totaltest.MixedHarness[*totaltest.InMemory]{Name: "in-memory", New: totaltest.NewInMemory},
		totaltest.MixedChecks{
			{
				Method: "Classify",
				Name:   "answers-for-the-empty-string",
				Claim:  "Classify answers for the empty string as readily as for any other",
				Run: func(tb testing.TB, s total.Mixed, fx totaltest.MixedFixture) {
					tb.Helper()
					// A subject that refused it would be total over "non-empty
					// strings", which is a different claim.
					got, err := s.Classify(tb.Context(), "")
					testkit.NoError(tb, err, "the empty string is in the domain")
					testkit.Equal(tb, got, "empty", "and is classified rather than refused")

					_, err = s.Classify(tb.Context(), fx.In())
					testkit.NoError(tb, err, "and so is anything else")
				},
			},
		},
	)
}
