// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package multiaggregatortest_test

import (
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

	multiaggregatortest.RunMultiAggregator(t,
		multiaggregatortest.MultiAggregatorHarness[*multiaggregatortest.InMemory]{
			Name: "in-memory",
			New: func() *multiaggregatortest.InMemory {
				s := multiaggregatortest.NewInMemory()
				s.Add(4)
				return s
			},
		},
		multiaggregatortest.MultiAggregatorChecks{
			{
				Method: "Stats",
				Name:   "reduces-to-both-numbers",
				Claim:  "Stats reduces the collection to both numbers",
				Run: func(tb testing.TB, s multiaggregator.MultiAggregator, fx multiaggregatortest.MultiAggregatorFixture) {
					tb.Helper()
					count, sum, err := s.Stats(tb.Context())
					testkit.NoError(tb, err, "reducing a healthy collection succeeds")
					testkit.Equal(tb, count, 1, "the count reports what the constructor put there")
					testkit.Equal(tb, sum, 4, "and the sum agrees with it")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestMultiAggregatorContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	multiaggregatortest.RunMultiAggregator(
		t,
		multiaggregatortest.MultiAggregatorHarness[*multiaggregatortest.InMemory]{
			Name: "in-memory",
			New:  multiaggregatortest.NewInMemory,
		},
		multiaggregatortest.MultiAggregatorSuite.Without(multiaggregatortest.MultiAggregatorSuite.Checks.Stats.Smoke()),
	)
}
