// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package readyourwritestest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/readyourwrites"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/readyourwrites/readyourwritestest"
)

// The generated contract, run against the in-memory subject.
//
// The mixin's own law spans more than one call and needs a reference to
// compare against, so it is the model tier's. What the suite tier states about
// the pair is the row below.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	readyourwritestest.RunMixed(
		t,
		readyourwritestest.MixedHarness[*readyourwritestest.InMemory]{
			Name: "in-memory",
			New:  readyourwritestest.NewInMemory,
		},
		readyourwritestest.MixedChecks{
			{
				Method: "Get",
				Name:   "reads-back-what-store-wrote",
				Claim:  "Get returns what Store wrote",
				Run: func(tb testing.TB, s readyourwrites.Mixed, fx readyourwritestest.MixedFixture) {
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
