// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package validated is the integration fixture for a type whose values a
// derived sample cannot guess.
//
// The other axes vary one thing about a declaration and check what gets
// stamped. This one varies nothing about the classification and everything
// about the subject: a real implementation, with real validation, held to the
// generated contract. It is where "the generator emitted something" and "the
// something it emitted is true of a working implementation" stop being the
// same claim.
//
// Account is the point. Its fields accept only some strings — an identifier
// shaped like a UUID, an address holding an `@` — and the sample a generator
// derives from a field's name and type is `test-id` and `test-email`, which the
// store refuses. A harness built on the derived value would report a
// conformance violation against a correct implementation, which is the failure
// mode that gets a suite switched off.
//
// [AccountDefaults] is the answer, and it is deliberately an ordinary function
// rather than a directive: the validation rules live in the author's head, and
// the convention that names them is the one the builder generator already
// reads. One function, two readers — builder seeds NewAccount with it, and the
// suite's fixture takes it over anything it could derive.
//
// Routing is declared once for the package rather than on each declaration:
// the builder, the double and the harness all belong beside each other, and a
// per-declaration directive is the same statement written three times.
//
//testkit:out validatedtest/ pkg=validatedtest
package validated

import (
	"context"
)

// Account is the payload, with fields that admit only some values.
//
// The builder exists so a check that needs a *variant* of a valid account —
// one field changed, the rest still acceptable — can say so in one line instead
// of restating every field and hoping it stays valid.
//
//testkit:builder
type Account struct {
	// ID is rejected unless it is UUID-shaped.
	ID string

	// Email is rejected unless it holds an `@`.
	Email string
}

// AccountDefaults returns an Account the store accepts.
//
// Hand-written, because nothing about the declaration says what `ID` admits.
// A generator deriving `test-id` is not wrong to try — it is right for the many
// types with no rules — and this is how a type with rules says so once, for
// every generator that needs a value of it.
func AccountDefaults() Account {
	return Account{
		ID:    "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		Email: "conformance@example.com",
	}
}

// Store is the subject the generated harness holds an implementation to.
//
//testkit:stub
//testkit:suite
type Store interface {
	// Put refuses an Account that does not validate.
	Put(ctx context.Context, a Account) error

	// Get reports the zero value alongside every error, which is the property
	// the reader's own check is about.
	Get(ctx context.Context, id string) (Account, error)
}
