// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package conservativetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/conservative"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/conservative/conservativetest"
)

// The counterpart to associative over the same two methods: there a fold is
// expected to move the total, here it is expected not to. Neither signature
// says which, which is why both are rows.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	conservativetest.RunMixed(t,
		conservativetest.MixedHarness[*conservativetest.InMemory]{Name: "in-memory", New: conservativetest.NewInMemory},
		conservativetest.MixedChecks{
			{
				Method: "Total",
				Name:   "transfer-conserves-the-sum",
				Claim:  "Total holds the conserved sum through a transfer",
				Run: func(tb testing.TB, s conservative.Mixed, fx conservativetest.MixedFixture) {
					tb.Helper()
					// Apply is a transfer: the conserved sum must still read as
					// it did at birth — a non-zero total is quantity minted
					// from nothing, the mixin's violation.
					testkit.NoError(tb, s.Apply(tb.Context(), fx.Delta()), "the transfer is applied")

					got, err := s.Total(tb.Context())
					testkit.NoError(tb, err, "the total is readable")
					testkit.Equal(tb, got, 0, "and the transfer conserved it")
				},
			},
		},
	)
}
