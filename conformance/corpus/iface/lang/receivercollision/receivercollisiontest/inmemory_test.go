// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package receivercollisiontest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/receivercollision"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/receivercollision/receivercollisiontest"
)

// Every method here names a parameter `s`, one at Session and one at string.
//
// The fixture keys on the name *and* the type, so the checks are handed a
// Session and a string rather than one value the other method could not take.
// The author's own rule is that the two agree: what Put stored under a
// session's identifier is what Get returns for it.
func TestStoreContract(t *testing.T) {
	t.Parallel()

	receivercollisiontest.RunStore(
		t,
		receivercollisiontest.StoreHarness[*receivercollisiontest.InMemory]{
			Name: "in-memory",
			New:  receivercollisiontest.NewInMemory,
		},
		receivercollisiontest.StoreChecks{
			{
				Method: "Get",
				Name:   "reads-back-the-stored-session",
				Claim:  "Get returns what Put stored under that identifier",
				Run: func(tb testing.TB, s receivercollision.Store, fx receivercollisiontest.StoreFixture) {
					tb.Helper()
					// The row draws both `s` parameters from the one fixture,
					// which is the whole point of the collision: they are
					// different fields because they are different types.
					stored := fx.Session()
					testkit.NoError(tb, s.Put(tb.Context(), stored), "the session is stored")

					got, err := s.Get(tb.Context(), stored.ID)
					testkit.NoError(tb, err, "a stored session is found by its own identifier")
					testkit.Equal(tb, got, stored, "and comes back whole")
				},
			},
		},
	)
}

// Suppression, against the same subject: what is under test is the harness
// declining what it was told to.
func TestStoreContractSuppression(t *testing.T) {
	t.Parallel()

	receivercollisiontest.RunStore(
		t,
		receivercollisiontest.StoreHarness[*receivercollisiontest.InMemory]{
			Name: "in-memory",
			New:  receivercollisiontest.NewInMemory,
		},
		receivercollisiontest.StoreSuite.Without(receivercollisiontest.StoreSuite.Checks.Touch.Smoke()),
	)
}
