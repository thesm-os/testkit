// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stableordertest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/stableorder"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/stableorder/stableordertest"
)

// The generated contract, run against the in-memory subject.
//
// The subject's stated order is key-ascending, which is the one fact no
// derivation invents — so the row states it.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	stableordertest.RunMixed(t,
		stableordertest.MixedHarness[*stableordertest.InMemory]{Name: "in-memory", New: stableordertest.NewInMemory},
		stableordertest.MixedChecks{
			{
				Method: "Items",
				Name:   "drains-in-key-order",
				Claim:  "Items yields what Add put in, in key order",
				Run: func(tb testing.TB, s stableorder.Mixed, fx stableordertest.MixedFixture) {
					tb.Helper()
					// Deliberately out of key order: with one element the
					// drain's ordering is unobservable, and a subject that
					// returned map order would pass.
					testkit.NoError(tb, s.Add(tb.Context(), stableorder.Value{Key: "zz", Body: "last"}),
						"an element is accepted")
					testkit.NoError(tb, s.Add(tb.Context(), stableorder.Value{Key: "aa", Body: "first"}),
						"and a second that sorts ahead of it")

					got, err := s.Items(tb.Context())
					testkit.NoError(tb, err, "the drain succeeds")
					testkit.Equal(tb, len(got), 2, "each append is one element")
					testkit.Equal(tb, got[0].Key, "aa", "and the drain is ordered rather than arbitrary")
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

	stableordertest.RunMixed(t,
		stableordertest.MixedHarness[*stableordertest.InMemory]{Name: "in-memory", New: stableordertest.NewInMemory},
		stableordertest.MixedSuite.Without(stableordertest.MixedSuite.Checks.Add.Smoke()),
	)
}
