// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package monotonicwritestest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/monotonicwrites"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/monotonicwrites/monotonicwritestest"
)

// The generated contract, run against the in-memory subject.
//
// The mixin's own law spans two sessions and needs a reference to compare
// against, so it is the model tier's. What the suite tier can state about the
// pair is the row below: a key Store wrote is a key Get answers for.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	monotonicwritestest.RunMixed(
		t,
		monotonicwritestest.MixedHarness[*monotonicwritestest.InMemory]{
			Name: "in-memory",
			New:  monotonicwritestest.NewInMemory,
		},
		monotonicwritestest.MixedChecks{
			{
				Method: "Get",
				Name:   "reads-back-what-store-wrote",
				Claim:  "Get returns what Store wrote",
				Run: func(tb testing.TB, s monotonicwrites.Mixed, fx monotonicwritestest.MixedFixture) {
					tb.Helper()
					written := fx.Value()
					// Store answers the state it wrote beside its error, which
					// is what makes this an answering writer rather than a
					// plain one.
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
