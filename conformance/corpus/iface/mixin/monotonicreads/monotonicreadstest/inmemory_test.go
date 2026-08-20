// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package monotonicreadstest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/monotonicreads"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/monotonicreads/monotonicreadstest"
)

// The generated contract, run against the in-memory subject.
//
// The mixin's own law spans two sessions and needs a reference to compare
// against, so it is the model tier's. What the suite tier can state about the
// pair is the row below: a key Store wrote is a key Get answers for.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	monotonicreadstest.RunMixed(
		t,
		monotonicreadstest.MixedHarness[*monotonicreadstest.InMemory]{
			Name: "in-memory",
			New:  monotonicreadstest.NewInMemory,
		},
		monotonicreadstest.MixedChecks{
			{
				Method: "Get",
				Name:   "reads-back-what-store-wrote",
				Claim:  "Get returns what Store wrote",
				Run: func(tb testing.TB, s monotonicreads.Mixed, fx monotonicreadstest.MixedFixture) {
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
