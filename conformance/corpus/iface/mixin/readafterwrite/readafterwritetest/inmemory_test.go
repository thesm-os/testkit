// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package readafterwritetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/readafterwrite"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/readafterwrite/readafterwritetest"
)

// readafterwrite is the model tier's — AUTO-READ-AFTER-WRITE states it — so the
// suite generates the signature family alone, even though the mixin names its
// partner through `write=Write`.
//
// Naming the partner is what makes the law bindable; stating it needs a
// reference to compare against, which is what separates the two tiers. The row
// below is the deterministic half: one write, one read.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	readafterwritetest.RunMixed(
		t,
		readafterwritetest.MixedHarness[*readafterwritetest.InMemory]{
			Name: "in-memory",
			New:  readafterwritetest.NewInMemory,
		},
		readafterwritetest.MixedChecks{
			{
				Method: "Read",
				Name:   "reads-back-what-write-wrote",
				Claim:  "Read returns what Write wrote",
				Run: func(tb testing.TB, s readafterwrite.Mixed, fx readafterwritetest.MixedFixture) {
					tb.Helper()
					testkit.NoError(tb, s.Write(tb.Context(), fx.Key(), fx.Value()), "the key is written")

					got, err := s.Read(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "a written key is found")
					testkit.Equal(tb, got, fx.Value(), "and carries what was written")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	readafterwritetest.RunMixed(
		t,
		readafterwritetest.MixedHarness[*readafterwritetest.InMemory]{
			Name: "in-memory",
			New:  readafterwritetest.NewInMemory,
		},
		readafterwritetest.MixedSuite.Without(readafterwritetest.MixedSuite.Checks.Write.Smoke()),
	)
}
