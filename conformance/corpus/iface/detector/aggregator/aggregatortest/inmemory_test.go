// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package aggregatortest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/aggregator"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/aggregator/aggregatortest"
)

// A method taking nothing after its context gets four checks, not five.
//
// Count returns a value beside an error, which is ordinarily enough to earn the
// "an error carries the zero value" check — but that check reaches the failure
// it is about by choosing an input that misses, and there is no input to vary.
// So the claim is written here instead, against a subject this package can
// break.
func TestAggregatorContract(t *testing.T) {
	t.Parallel()

	aggregatortest.RunAggregator(t,
		aggregatortest.AggregatorHarness[*aggregatortest.InMemory]{
			Name: "in-memory",
			// Aggregator declares no writer, so nothing is derived to seed
			// through and the seed is the constructor's.
			New: func() *aggregatortest.InMemory {
				s := aggregatortest.NewInMemory()
				s.Add("seeded")
				return s
			},
		},
		aggregatortest.AggregatorChecks{
			{
				Method: "Count",
				Name:   "counts-what-it-holds",
				Claim:  "Count counts what the collection holds",
				Run: func(tb testing.TB, s aggregator.Aggregator, fx aggregatortest.AggregatorFixture) {
					tb.Helper()
					got, err := s.Count(tb.Context())
					testkit.NoError(tb, err, "counting a healthy collection succeeds")
					testkit.Equal(tb, got, 1, "and reports what the constructor put there")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestAggregatorContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	aggregatortest.RunAggregator(t,
		aggregatortest.AggregatorHarness[*aggregatortest.InMemory]{Name: "in-memory", New: aggregatortest.NewInMemory},
		aggregatortest.AggregatorSuite.Without(aggregatortest.AggregatorSuite.Checks.Count.Smoke()),
	)
}
