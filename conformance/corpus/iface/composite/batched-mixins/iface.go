// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package batchedmixins is the fixture for several mixin names on one line.
//
// The mixin directive declares AllowExtraPositional, so every positional
// argument after the first is another mixin name. That is the whole reason for
// the batching form recorded in docs/adr/0016: a method carrying six
// properties should cost one comment line rather than six.
//
// Nothing else in the corpus exercises it. Every other mixin fixture writes a
// single name, which parses identically whether or not extra positionals are
// permitted — so a regression that dropped AllowExtraPositional would leave
// the whole corpus green and only fail in consumer code.
//
// The interface also pairs a batched line with a parameterised one, because
// the schema permits key-value pairs only when exactly one name is given. The
// two forms cannot be combined, and having both here makes that boundary
// visible in one place.
package batchedmixins

import "context"

// Batched is the fixture interface.
//
//testkit:out batchedmixinstest/ pkg=batchedmixinstest
//testkit:stub
//testkit:suite
type Batched interface {
	// Put carries three mixins on one line. Bare tokens are read as further
	// names, never as parameters, which is why none of these may take one.
	//testkit:mixin idempotent concurrent sideeffect
	Put(ctx context.Context, key, value string) error

	// Read carries a single parameterised mixin. Parameters are permitted
	// only with exactly one name — with several the owning mixin would be
	// ambiguous — so this cannot be folded into the line above.
	//testkit:mixin readafterwrite write=Put
	Read(ctx context.Context, key string) (string, error)

	// List stacks a batched line and a parameterised one on the same method,
	// which the schema allows because each line names its own owner.
	//testkit:mixin cacheable pure
	//testkit:mixin bounded limit=50
	List(ctx context.Context) ([]string, error)
}
