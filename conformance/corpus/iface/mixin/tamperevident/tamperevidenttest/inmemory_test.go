// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tamperevidenttest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/tamperevident"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/tamperevident/tamperevidenttest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	tamperevidenttest.RunMixed(
		t,
		tamperevidenttest.MixedHarness[*tamperevidenttest.InMemory]{
			Name: "in-memory",
			New:  tamperevidenttest.NewInMemory,
		},
		tamperevidenttest.MixedChecks{
			{
				Method: "Verify",
				Name:   "detects-an-alteration",
				Claim:  "Verify detects a value altered behind its back",
				Run: func(tb testing.TB, s tamperevident.Mixed, fx tamperevidenttest.MixedFixture) {
					tb.Helper()
					// Corrupt reaches past the interface, so there has to be
					// something stored for it to reach.
					testkit.NoError(tb, s.Store(tb.Context(), fx.Body()), "a value is stored")

					testkit.NoError(tb, s.Verify(tb.Context()), "an untouched value verifies")
					testkit.NoError(tb, s.Corrupt(tb.Context()), "the bytes are altered")
					testkit.Error(tb, s.Verify(tb.Context()),
						"and the alteration is detected rather than served")
				},
			},
		},
	)
}
