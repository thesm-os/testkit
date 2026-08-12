// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package multiaggregatortest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/multiaggregator"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/multiaggregator/multiaggregatortest"
)

// Two value slots beside an error and still four checks, not five.
//
// The zero-value check would have more to say here than for the single
// aggregator — two slots can disagree with the error independently — and it is
// still not generated, for the same reason: Stats takes nothing, so the harness
// has no input to choose and could only demand failure from a correct
// implementation. Both halves of the claim are written below instead.
func TestMultiAggregatorContract(t *testing.T) {
	t.Parallel()

	multiaggregatortest.AssertMultiAggregatorContract(t,
		multiaggregatortest.MultiAggregatorModel(),
		multiaggregatortest.MultiAggregatorSubject("in-memory",
			func() multiaggregator.MultiAggregator {
				return multiaggregatortest.NewInMemory()
			}),
		multiaggregatortest.MultiAggregatorSeed(
			func(_ context.Context, subject multiaggregator.MultiAggregator) error {
				// A seed may reach for the concrete subject: it runs before the
				// double wraps it and sees what the factory made. A check may
				// not.
				subject.(*multiaggregatortest.InMemory).Add(4)
				return nil
			},
		),
		multiaggregatortest.MultiAggregatorOnStats("reduces the collection to both numbers", func(
			tb testing.TB, subject multiaggregator.MultiAggregator,
		) {
			tb.Helper()
			count, sum, err := subject.Stats(tb.Context())
			testkit.NoError(tb, err, "reducing a healthy collection succeeds")
			testkit.Equal(tb, count, 1, "the count reports what the seed put there")
			testkit.Equal(tb, sum, 4, "and the sum agrees with it")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMultiAggregatorContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	multiaggregatortest.AssertMultiAggregatorContract(t,
		multiaggregatortest.MultiAggregatorSubject("in-memory",
			func() multiaggregator.MultiAggregator {
				return multiaggregatortest.NewInMemory()
			}),
		multiaggregatortest.MultiAggregatorWithout("Stats/smoke"),
		multiaggregatortest.MultiAggregatorWithoutDouble(),
	)
}
