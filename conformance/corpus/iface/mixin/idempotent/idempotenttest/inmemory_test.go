// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package idempotenttest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/idempotent"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/idempotent/idempotenttest"
)

// idempotent is the model tier's — AUTO-IDEMPOTENT-WRITE states it — so the
// suite generates the signature family plus the repeat probe.
//
// That is the assignment working rather than a gap: the repeat write and the
// single write return the same thing, so no one call can tell them apart.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	idempotenttest.RunMixed(t,
		idempotenttest.MixedHarness[*idempotenttest.InMemory]{Name: "in-memory", New: idempotenttest.NewInMemory},
		idempotenttest.MixedChecks{
			{
				Method: "Read",
				Name:   "reads-back-what-put-wrote",
				Claim:  "Read returns what Put wrote",
				Run: func(tb testing.TB, s idempotent.Mixed, fx idempotenttest.MixedFixture) {
					tb.Helper()
					testkit.NoError(tb, s.Put(tb.Context(), fx.Key(), fx.Value()), "the key is written")

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

	idempotenttest.RunMixed(t,
		idempotenttest.MixedHarness[*idempotenttest.InMemory]{Name: "in-memory", New: idempotenttest.NewInMemory},
		idempotenttest.MixedSuite.Without(idempotenttest.MixedSuite.Checks.Put.Smoke()),
	)
}
