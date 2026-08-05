// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package deleteremoves is the mixin-axis fixture for the deleteremoves mixin, which
// declares that a delete makes the key unreadable afterwards.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package deleteremoves

import (
	"context"
)

// Mixed is the fixture interface.
type Mixed interface {
	// Delete must make the key unreadable. Without Read the law has no
	// observation to make, and without Put there is nothing to delete.
	//testkit:mixin deleteremoves
	Delete(ctx context.Context, key string) error

	// Put establishes the key the delete removes.
	Put(ctx context.Context, key, value string) error

	// Read observes the removal.
	Read(ctx context.Context, key string) (string, error)
}
