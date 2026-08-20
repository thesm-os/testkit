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

	leakfreetest.RunMixed(t,
		leakfreetest.MixedHarness[*leakfreetest.InMemory]{Name: "in-memory", New: leakfreetest.NewInMemory},
		leakfreetest.MixedChecks{
			{
				Method: "Release",
				Name:   "release-without-acquire-is-refused",
				Claim:  "Release refuses a release nothing acquired",
				Run: func(tb testing.TB, s leakfree.Mixed, fx leakfreetest.MixedFixture) {
					tb.Helper()
					// The cycle, stated once: acquire, release, and the balance
					// is back to zero. A second release then has nothing to give
					// back, which is what keeps the count from going negative
					// and hiding an asymmetry.
					testkit.NoError(tb, s.Acquire(tb.Context()), "the resource is available")
					testkit.NoError(tb, s.Release(tb.Context()), "and giving it back succeeds")

					held, err := s.Outstanding(tb.Context())
					testkit.NoError(tb, err, "the balance is readable")
					testkit.Equal(tb, held, 0, "a completed cycle leaves nothing outstanding")

					testkit.Error(tb, s.Release(tb.Context()),
						"a release with nothing held is refused rather than counted")
				},
			},
		},
	)
}
