// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package aggregatortest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/aggregator"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/aggregator/aggregatortest"
)

// A method taking nothing after its context gets four checks, not five.
//
// Count returns a value beside an error, which is ordinarily enough to earn the
// "an error carries the zero value" check — but that check reaches the failure
// it is about by choosing an input that misses, and there is no input. Generated
// anyway it would demand failure from a correct implementation, so the claim is
// written here instead, against a subject this package can break.
func TestAggregatorContract(t *testing.T) {
	t.Parallel()

	aggregatortest.AssertAggregatorContract(t,
		aggregatortest.AggregatorSubject("in-memory", func() aggregator.Aggregator {
			return aggregatortest.NewInMemory()
		}),
		aggregatortest.AggregatorSeed(func(_ context.Context, subject aggregator.Aggregator) error {
			// Aggregator declares no writer, so nothing is derived. A seed may
			// reach for the concrete subject: it runs before the double wraps
			// it and sees what the factory made. A check may not.
			subject.(*aggregatortest.InMemory).Add("seeded")
			return nil
		}),
		aggregatortest.AggregatorOnCount("counts what the collection holds", func(
			tb testing.TB, subject aggregator.Aggregator,
		) {
			tb.Helper()
			got, err := subject.Count(tb.Context())
			testkit.NoError(tb, err, "counting a healthy collection succeeds")
			testkit.Equal(tb, got, 1, "and reports what the seed put there")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestAggregatorContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	aggregatortest.AssertAggregatorContract(t,
		aggregatortest.AggregatorSubject("in-memory", func() aggregator.Aggregator {
			return aggregatortest.NewInMemory()
		}),
		aggregatortest.AggregatorWithout("Count/smoke"),
		aggregatortest.AggregatorWithoutDouble(),
	)
}
