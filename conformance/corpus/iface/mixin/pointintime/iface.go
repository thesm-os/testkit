// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package pointintime is the mixin-axis fixture for the pointintime mixin, which
// declares that two reads of one key agree even when a write lands
// between them.
//
// The interface carries a write and a read because the claim relates them:
// a read alone has nothing to be consistent with, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package pointintime

import (
	"context"
)

// Value is the payload the store holds.
type Value struct{ Key, Body string }

// Mixed is the fixture interface.
//
//testkit:out pointintimetest/ pkg=pointintimetest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Store is the write the read is judged against.
	Store(ctx context.Context, v Value) error

	// Get reads a key. Stronger than cacheable, which permits the second read to
	// observe a concurrent write.
	//testkit:mixin pointintime
	Get(ctx context.Context, key string) (Value, error)
}
