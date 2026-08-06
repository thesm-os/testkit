// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package cacheable is the mixin-axis fixture for the cacheable mixin, which
// declares that repeated reads may be served from a cache without changing the answer.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package cacheable

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out cacheabletest/ pkg=cacheabletest
//testkit:stub
type Mixed interface {
	// Get may be cached. The law is that caching is unobservable: two reads
	// agree whether or not the second was served from memory.
	//testkit:mixin cacheable
	Get(ctx context.Context, key string) (string, error)
}
