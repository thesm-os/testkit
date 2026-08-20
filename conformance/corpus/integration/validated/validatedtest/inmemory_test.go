// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package validatedtest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/integration/validated"
	"go.thesmos.sh/testkit/conformance/corpus/integration/validated/validatedtest"
)

// Three generators over one declaration, wired by a consumer in one statement.
//
// The suite supplies the contract and the subjects. The stub supplies the
// second run, wrapping each subject so anything the wrapper fails that the
// subject passes is the double lying. The builder supplies the variant the
// custom check below needs — one field changed, the rest still acceptable.
//
// What binds them is AccountDefaults: builder seeds NewAccount with it and the
// suite's fixture takes it over anything it could derive, so a team states the
// valid shape of an Account once and every generator that needs one uses it.
func TestStoreContract(t *testing.T) {
	t.Parallel()

	validatedtest.RunStore(t,
		validatedtest.StoreHarness[*validatedtest.InMemory]{Name: "in-memory", New: validatedtest.NewInMemory},
		validatedtest.StoreChecks{
			{
				Method: "Get",
				Name:   "reads-back-the-stored-account",
				Claim:  "Get returns the account Put stored",
				Run: func(tb testing.TB, s validated.Store, fx validatedtest.StoreFixture) {
					tb.Helper()
					// A domain rule no classification implies. It runs against
					// every subject and through the double, from this one row.
					want := validated.AccountDefaults()
					testkit.NoError(tb, s.Put(tb.Context(), want), "a valid account stores")

					got, err := s.Get(tb.Context(), want.ID)
					testkit.NoError(tb, err, "a stored account is found by its own identifier")
					testkit.Equal(tb, got, want, "and comes back whole")
				},
			},
			{
				Method: "Put",
				Name:   "refuses-an-address-with-no-at",
				Claim:  "Put refuses an address with no @",
				Run: func(tb testing.TB, s validated.Store, fx validatedtest.StoreFixture) {
					tb.Helper()
					// Built from the valid account rather than written out: a
					// literal restating every field goes stale the moment one is
					// added, and would then be refused for the wrong reason.
					bad := validatedtest.NewAccountFrom(fx.Account()).WithEmail("no-at-sign").Build()
					testkit.ErrorIs(tb, s.Put(tb.Context(), bad), validatedtest.ErrInvalid,
						"Put must refuse an account whose email is not one")
				},
			},
		},
	)
}
