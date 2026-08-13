// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package total is the mixin-axis fixture for the total mixin, which
// declares that a callable answers for every input in a named domain rather
// than failing on some of them.
//
// Two methods, differing only in whether they can report a failure. The claim
// is about what a single call returns for a hostile input rather than about
// how two calls relate, so one would state it — but the law's closure threads
// an error, and a method with no error to thread is the shape that made the
// binder emit a call the compiler refuses. A total function that cannot fail
// is the strongest form of the claim, and the fixture the tier had none of.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package total

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out totaltest/ pkg=totaltest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Classify answers for any input in the declared domain.
	//testkit:mixin total domain=strings
	Classify(ctx context.Context, in string) (string, error)

	// Normalize answers for any input and has no way to report otherwise.
	//testkit:mixin total domain=strings
	Normalize(in string) string
}
