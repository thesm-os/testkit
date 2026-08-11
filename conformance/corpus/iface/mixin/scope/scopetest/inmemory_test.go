// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package scopetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/scope"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/scope/scopetest"
)

// scope is the suite tier's under ADR-0018 and generates no check, because what
// it needs is a value no run can invent.
//
// `//testkit:mixin scope name=tenant` names the scope, and the RFC's open list
// has it right: the check wants an *authorised* context and an unauthorised
// sentinel, and neither is derivable — an authorisation is a fact about a
// deployment, not about a signature.
//
// Isolation between scopes is the derivable half, and it is `partition`'s check
// under another name: same key, two scopes, read one back. This fixture carries
// no `axis=`, so nothing is generated for it here.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	fixture := scopetest.DefaultMixedFixture()

	scopetest.AssertMixedContract(t,
		scopetest.MixedSubject("in-memory", func() scope.Mixed {
			return scopetest.NewInMemory()
		}),
		scopetest.MixedOnGet("reads within its scope", func(
			tb testing.TB, subject scope.Mixed, sc, key string,
		) {
			tb.Helper()
			testkit.NoError(tb, subject.Set(tb.Context(), sc, key, fixture.Value),
				"writing within a scope succeeds")
			got, err := subject.Get(tb.Context(), sc, key)
			testkit.NoError(tb, err, "and reading it back succeeds")
			testkit.Equal(tb, got, fixture.Value, "carrying what was written")
		}),
	)
}

// One scope does not see another's writes. A store hashing the scope and key
// together satisfies every generated check and leaks across tenants.
func TestScopesDoNotLeak(t *testing.T) {
	t.Parallel()

	s := scopetest.NewInMemory()
	ctx := t.Context()

	testkit.NoError(t, s.Set(ctx, "a", "k", "one"), "writing to a succeeds")
	testkit.NoError(t, s.Set(ctx, "b", "k", "two"), "writing to b succeeds")

	got, err := s.Get(ctx, "a", "k")
	testkit.NoError(t, err, "reading a succeeds")
	testkit.Equal(t, got, "one", "and returns a's value rather than b's")

	_, err = s.Get(ctx, "c", "k")
	testkit.ErrorIs(t, err, scopetest.ErrNotFound,
		"a scope nothing was written to holds nothing")
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	scopetest.AssertMixedContract(t,
		scopetest.MixedSubject("in-memory", func() scope.Mixed {
			return scopetest.NewInMemory()
		}),
		scopetest.MixedWithout("Set/smoke"),
		scopetest.MixedWithoutDouble(),
	)
}
