// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package writesfollowreadstest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/writesfollowreads"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/writesfollowreads/writesfollowreadstest"
)

// The generated contract, run against the in-memory subject.
//
// The session-ordering law needs a reference to compare against, so it is the
// model tier's. What the suite tier states about the pair is the row below.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	writesfollowreadstest.RunMixed(
		t,
		writesfollowreadstest.MixedHarness[*writesfollowreadstest.InMemory]{
			Name: "in-memory",
			New:  writesfollowreadstest.NewInMemory,
		},
		writesfollowreadstest.MixedChecks{
			{
				Method: "Get",
				Name:   "reads-back-what-store-wrote",
				Claim:  "Get returns what Store wrote",
				Run: func(tb testing.TB, s writesfollowreads.Mixed, fx writesfollowreadstest.MixedFixture) {
					tb.Helper()
					written := fx.Value()
					// Store answers the state it wrote beside its error.
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
