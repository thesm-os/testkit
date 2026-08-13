// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package totaltest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/total"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/total/totaltest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	totaltest.AssertMixedContract(t,
		totaltest.MixedModel(),
		totaltest.MixedSubject("in-memory", func() total.Mixed {
			return totaltest.NewInMemory()
		}),
		totaltest.MixedOnClassify("answers for the empty string as readily as for any other", func(
			tb testing.TB, subject total.Mixed, in string,
		) {
			tb.Helper()
			// The edge of the domain: a subject that refused it would be
			// total over "non-empty strings", which is a different claim.
			got, err := subject.Classify(tb.Context(), "")
			testkit.NoError(tb, err, "the empty string is in the domain")
			testkit.Equal(tb, got, "empty", "and is classified rather than refused")
			_ = in
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	totaltest.AssertMixedContract(t,
		totaltest.MixedSubject("in-memory", func() total.Mixed {
			return totaltest.NewInMemory()
		}),
		totaltest.MixedWithoutDouble(),
	)
}
