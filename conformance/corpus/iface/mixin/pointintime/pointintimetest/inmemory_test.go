// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package pointintimetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/pointintime"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/pointintime/pointintimetest"
)

// The generated contract, run against the in-memory subject.
//
// The snapshot claim itself needs a reference to compare against, so it is the
// model tier's. What the suite tier states about the pair is the row below.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	pointintimetest.RunMixed(t,
		pointintimetest.MixedHarness[*pointintimetest.InMemory]{Name: "in-memory", New: pointintimetest.NewInMemory},
		pointintimetest.MixedChecks{
			{
				Method: "Get",
				Name:   "reads-back-what-store-wrote",
				Claim:  "Get returns what Store wrote",
				Run: func(tb testing.TB, s pointintime.Mixed, fx pointintimetest.MixedFixture) {
					tb.Helper()
					written := fx.Value()
					testkit.NoError(tb, s.Store(tb.Context(), written), "the value is stored")

					got, err := s.Get(tb.Context(), written.Key)
					testkit.NoError(tb, err, "the written key is present")
					testkit.Equal(tb, got.Key, written.Key,
						"and Get answers under the key it was stored with")
				},
			},
		},
	)
}
