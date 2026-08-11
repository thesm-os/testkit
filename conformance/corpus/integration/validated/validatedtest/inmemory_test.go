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

	validatedtest.AssertStoreContract(t,
		validatedtest.StoreSubject("in-memory", func() validated.Store {
			return validatedtest.NewInMemory()
		}),

		// A domain rule no classification implies. It runs against every
		// subject and through the double, from this one statement.
		// The seed writes AccountDefaults, whose ID is UUID-shaped; the derived
		// key the reader's checks are handed is "test-id". They cannot agree,
		// so the hit path is one only a check reading the seeded account's own
		// identifier reaches — which is the shape of every read-after-write
		// claim, and the reason the classification that states it is the model
		// tier's rather than something derivable here.
		validatedtest.StoreOnGet("returns the account Put stored", func(
			tb testing.TB, subject validated.Store, id string,
		) {
			tb.Helper()
			want := validated.AccountDefaults()
			got, err := subject.Get(tb.Context(), want.ID)
			testkit.NoError(tb, err, "a stored account is found by its own identifier")
			testkit.Equal(tb, got, want, "and comes back whole")
		}),
		validatedtest.StoreOnPut("refuses an address with no @",
			func(tb testing.TB, subject validated.Store, a validated.Account) {
				tb.Helper()
				// Built from the valid account rather than written out: a
				// literal restating every field goes stale the moment one is
				// added, and would then be refused for the wrong reason.
				bad := validatedtest.NewAccountFrom(a).WithEmail("no-at-sign").Build()
				testkit.ErrorIs(tb, subject.Put(tb.Context(), bad), validatedtest.ErrInvalid,
					"Put must refuse an account whose email is not one")
			}),
	)
}
