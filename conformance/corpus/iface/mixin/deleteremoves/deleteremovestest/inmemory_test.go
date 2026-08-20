// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package deleteremovestest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/deleteremoves"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/deleteremoves/deleteremovestest"
)

// deleteremoves is the model tier's — AUTO-DELETE-RETURNS-NOT-FOUND states it —
// so the suite generates the signature family alone, even though eidos now lets
// the mixin name its reader through `read=Read`.
//
// Naming the partner is what makes the law bindable; stating it needs a
// reference to compare against, which is the line ADR-0018 draws. The row
// below is the deterministic half.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	deleteremovestest.RunMixed(
		t,
		deleteremovestest.MixedHarness[*deleteremovestest.InMemory]{
			Name: "in-memory",
			New:  deleteremovestest.NewInMemory,
		},
		deleteremovestest.MixedChecks{
			{
				Method: "Read",
				Name:   "delete-removes-what-put-wrote",
				Claim:  "Read reports the declared sentinel once Delete has run",
				Run: func(tb testing.TB, s deleteremoves.Mixed, fx deleteremovestest.MixedFixture) {
					tb.Helper()
					testkit.NoError(tb, s.Put(tb.Context(), fx.Key(), fx.Value()), "the key is written")

					got, err := s.Read(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "a written key is found")
					testkit.Equal(tb, got, fx.Value(), "and carries what was written")

					testkit.NoError(tb, s.Delete(tb.Context(), fx.Key()), "the key is removed")

					_, err = s.Read(tb.Context(), fx.Key())
					testkit.ErrorIs(tb, err, deleteremoves.ErrGone,
						"and reads after it report the declared sentinel")
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

	deleteremovestest.RunMixed(
		t,
		deleteremovestest.MixedHarness[*deleteremovestest.InMemory]{
			Name: "in-memory",
			New:  deleteremovestest.NewInMemory,
		},
		deleteremovestest.MixedSuite.Without(deleteremovestest.MixedSuite.Checks.Delete.Smoke()),
	)
}
