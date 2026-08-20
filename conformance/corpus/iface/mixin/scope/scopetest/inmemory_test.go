// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package scopetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/scope"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/scope/scopetest"
)

// scope generates no check of its own, because what it needs is a value no run
// can invent.
//
// `//testkit:mixin scope name=tenant` names the scope, and the RFC's open list
// has it right: the check wants an *authorised* context and an unauthorised
// sentinel, and neither is derivable — an authorisation is a fact about a
// deployment, not about a signature.
//
// Isolation between scopes is the derivable half, and it is `partition`'s check
// under another name: same key, two scopes, read one back. This fixture carries
// no `axis=`, so nothing is generated for it here and the row states the
// round trip instead.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	scopetest.RunMixed(t,
		scopetest.MixedHarness[*scopetest.InMemory]{Name: "in-memory", New: scopetest.NewInMemory},
		scopetest.MixedChecks{
			{
				Method: "Get",
				Name:   "reads-within-its-scope",
				Claim:  "Get reads within its scope",
				Run: func(tb testing.TB, s scope.Mixed, fx scopetest.MixedFixture) {
					tb.Helper()
					testkit.NoError(tb, s.Set(tb.Context(), fx.Scope(), fx.Key(), fx.Value()),
						"writing within a scope succeeds")

					got, err := s.Get(tb.Context(), fx.Scope(), fx.Key())
					testkit.NoError(tb, err, "and reading it back succeeds")
					testkit.Equal(tb, got, fx.Value(), "carrying what was written")
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

	scopetest.RunMixed(t,
		scopetest.MixedHarness[*scopetest.InMemory]{Name: "in-memory", New: scopetest.NewInMemory},
		scopetest.MixedSuite.Without(scopetest.MixedSuite.Checks.Set.Smoke()),
	)
}
