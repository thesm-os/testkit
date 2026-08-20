// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package causaltest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/causal"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/causal/causaltest"
)

// The generated contract, run against the in-memory subject.
//
// Causal consistency is a claim about an order across sessions and needs a
// reference to compare against, so it is the model tier's. What the suite tier
// states about the pair is the row below.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	causaltest.RunMixed(t,
		causaltest.MixedHarness[*causaltest.InMemory]{Name: "in-memory", New: causaltest.NewInMemory},
		causaltest.MixedChecks{
			{
				Method: "Get",
				Name:   "reads-back-what-store-wrote",
				Claim:  "Get returns what Store wrote",
				Run: func(tb testing.TB, s causal.Mixed, fx causaltest.MixedFixture) {
					tb.Helper()
					written := fx.Value()
					stored, err := s.Store(tb.Context(), written)
					testkit.NoError(tb, err, "the value is stored")
					testkit.Equal(tb, stored.Key, written.Key, "under the key it was given")

					got, err := s.Get(tb.Context(), written.Key)
					testkit.NoError(tb, err, "the written key is present")
					testkit.Equal(tb, got.Key, written.Key,
						"and Get answers under the key it was stored with")
				},
			},
		},
	)
}
