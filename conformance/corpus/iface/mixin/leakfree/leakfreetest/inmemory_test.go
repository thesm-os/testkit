// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package leakfreetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/leakfree"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/leakfree/leakfreetest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	leakfreetest.AssertMixedContract(t,
		leakfreetest.MixedModel(),
		leakfreetest.MixedSubject("in-memory", func() leakfree.Mixed {
			return leakfreetest.NewInMemory()
		}),
		leakfreetest.MixedOnRelease("refuses a release nothing acquired", func(
			tb testing.TB, subject leakfree.Mixed,
		) {
			tb.Helper()
			// The cycle, stated once: acquire, release, and the balance is
			// back to zero. A second release then has nothing to give back,
			// which is what keeps the count from going negative and hiding
			// an asymmetry.
			testkit.NoError(tb, subject.Acquire(tb.Context()), "the resource is available")
			testkit.NoError(tb, subject.Release(tb.Context()), "and giving it back succeeds")

			held, err := subject.Outstanding(tb.Context())
			testkit.NoError(tb, err, "the balance is readable")
			testkit.Equal(tb, held, 0, "a completed cycle leaves nothing outstanding")

			testkit.Error(tb, subject.Release(tb.Context()),
				"a release with nothing held is refused rather than counted")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	leakfreetest.AssertMixedContract(t,
		leakfreetest.MixedSubject("in-memory", func() leakfree.Mixed {
			return leakfreetest.NewInMemory()
		}),
		leakfreetest.MixedWithoutDouble(),
	)
}
