// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package partition is the mixin-axis fixture for the partition mixin, which
// declares that state is isolated per partition key.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package partition

import (
	"context"
)

// Mixed is the fixture interface.
type Mixed interface {
	// Put is partitioned: two keys in different partitions never interfere.
	// The law needs a partition parameter distinct from the key, or there is
	// nothing to isolate by.
	//testkit:mixin partition
	Put(ctx context.Context, partition, key, value string) error

	// Read observes isolation by reading the same key from another partition.
	Read(ctx context.Context, partition, key string) (string, error)
}
