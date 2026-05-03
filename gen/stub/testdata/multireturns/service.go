// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package multireturns

import "context"

//go:generate testkit stub -o servicetest/service_stub.gen.go Service

// Lease represents a held lock.
type Lease struct {
	ID  string
	TTL int
}

// Item is a stored value.
type Item struct {
	Key   string
	Value string
}

// Service exercises unnamed multi-non-error returns. The generator must
// produce distinct field names (Result, Result2, etc.) when return values
// are not named in the interface.
type Service interface {
	// Checkout returns an item and its lease. Two non-error returns + error.
	Checkout(ctx context.Context, key string) (Item, Lease, error)
	// Peek returns two items. Two identical-type non-error returns.
	Peek(ctx context.Context) (Item, Item, error)
	// Stats returns multiple scalars. Three non-error returns + error.
	Stats(ctx context.Context) (int, int, string, error)
}
