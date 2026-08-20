// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package concurrentreaderstest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/concurrentreaders"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/concurrentreaders/concurrentreaderstest"
)

// concurrentreaders is the suite tier's under ADR-0018, and its check is still
// not generated — the header records the gap.
//
// Readers that do not corrupt one another is observable only under the race
// detector, and `make check` runs `mod`, `lint`, `test`, `coverage` and
// `branch` — not `test race`. A generated check asserting nothing under the
// default gate would be decoration that reads as coverage, so the claim is made
// here where its conditions are visible.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	concurrentreaderstest.RunMixed(
		t,
		concurrentreaderstest.MixedHarness[*concurrentreaderstest.InMemory]{
			Name: "in-memory",
			New:  concurrentreaderstest.NewInMemory,
		},
		concurrentreaderstest.MixedChecks{
			{
				Method: "Get",
				Name:   "reads-back-what-put-wrote",
				Claim:  "Get returns what Put wrote",
				Run: func(tb testing.TB, s concurrentreaders.Mixed, fx concurrentreaderstest.MixedFixture) {
					tb.Helper()
					testkit.NoError(tb, s.Put(tb.Context(), fx.Key(), fx.Value()), "the key is written")

					got, err := s.Get(tb.Context(), fx.Key())
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

	concurrentreaderstest.RunMixed(
		t,
		concurrentreaderstest.MixedHarness[*concurrentreaderstest.InMemory]{
			Name: "in-memory",
			New:  concurrentreaderstest.NewInMemory,
		},
		concurrentreaderstest.MixedSuite.Without(concurrentreaderstest.MixedSuite.Checks.Put.Smoke()),
	)
}
