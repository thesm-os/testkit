// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package defaultonerrortest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/defaultonerror"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/defaultonerror/defaultonerrortest"
)

// The generated contract, run against the in-memory subject.
//
// Store is classified writer, so Get's miss check knows what a miss means
// here. What a key that WAS written reads back as is the row below.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	defaultonerrortest.RunMixed(
		t,
		defaultonerrortest.MixedHarness[*defaultonerrortest.InMemory]{
			Name: "in-memory",
			New:  defaultonerrortest.NewInMemory,
		},
		defaultonerrortest.MixedChecks{
			{
				Method: "Get",
				Name:   "reads-back-what-store-wrote",
				Claim:  "Get returns what Store wrote",
				Run: func(tb testing.TB, s defaultonerror.Mixed, fx defaultonerrortest.MixedFixture) {
					tb.Helper()
					written := fx.Value()
					testkit.NoError(tb, s.Store(tb.Context(), written), "the value is stored")

					got, err := s.Get(tb.Context(), written.Key)
					testkit.NoError(tb, err, "the written key is present")
					testkit.Equal(tb, got, written, "and answers with what was stored")
				},
			},
		},
	)
}
