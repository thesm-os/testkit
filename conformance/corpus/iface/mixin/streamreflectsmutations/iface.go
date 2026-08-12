// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package streamreflectsmutations is the mixin-axis fixture for the streamreflectsmutations mixin, which
// declares that a stream opened after a write observes that write.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package streamreflectsmutations

import (
	"context"
	"iter"
)

// Mixed is the fixture interface.
//
//testkit:out streamreflectsmutationstest/ pkg=streamreflectsmutationstest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Stream must reflect writes that preceded it. The law needs both a
	// mutation and a stream, which a read-only interface cannot provide.
	//testkit:mixin streamreflectsmutations mutate=Add delete=Remove
	Stream(ctx context.Context) iter.Seq2[string, error]

	// Add is the mutation the stream must reflect.
	Add(ctx context.Context, item string) error

	// Remove is the mutation's inverse, which the stream must also reflect —
	// and the half that lets AUTO-STREAM-REFLECTS-MUTATIONS clean up after
	// its own put.
	Remove(ctx context.Context, item string) error
}
