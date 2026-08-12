// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package retrysucceeds is the mixin-axis fixture for the retrysucceeds mixin, which
// declares that a transient failure succeeds on a later attempt.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package retrysucceeds

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out retrysucceedstest/ pkg=retrysucceedstest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Call fails transiently before succeeding. The law drives it repeatedly,
	// so a method that either always works or never does cannot host it.
	//testkit:mixin retrysucceeds
	//testkit:fault retry=3
	Call(ctx context.Context, key string) error

	// Attempts reports how many tries the subject has seen, which is how the
	// law distinguishes a retry from a first success.
	Attempts(ctx context.Context) (int, error)
}
