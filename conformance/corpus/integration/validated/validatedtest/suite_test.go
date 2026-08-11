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
			return validated.NewInMemory()
		}),

		// A domain rule no classification implies. It runs against every
		// subject and through the double, from this one statement.
		validatedtest.StoreOnPut("refuses an address with no @",
			func(tb testing.TB, subject validated.Store, a validated.Account) {
				tb.Helper()
				// Built from the valid account rather than written out: a
				// literal restating every field goes stale the moment one is
				// added, and would then be refused for the wrong reason.
				bad := validatedtest.NewAccountFrom(a).WithEmail("no-at-sign").Build()
				testkit.ErrorIs(tb, subject.Put(tb.Context(), bad), validated.ErrInvalid,
					"Put must refuse an account whose email is not one")
			}),
	)
}
