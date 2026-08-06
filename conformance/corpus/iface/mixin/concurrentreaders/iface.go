// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package concurrentreaders is the mixin-axis fixture for the concurrentreaders mixin, which
// declares that concurrent reads stay safe while a writer runs.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package concurrentreaders

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out concurrentreaderstest/ pkg=concurrentreaderstest
//testkit:stub
type Mixed interface {
	// Get is driven concurrently against Put. Reads alone cannot violate
	// this, so the interface has to carry the writer that contends with them.
	//testkit:mixin concurrentreaders
	Get(ctx context.Context, key string) (string, error)

	// Put is the contending writer.
	Put(ctx context.Context, key, value string) error
}
