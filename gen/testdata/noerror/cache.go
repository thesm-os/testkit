// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package noerror

import "context"

//go:generate testkit stub -o cachetest/cache_stub.gen.go Cache
//go:generate testkit suite -o cachetest/cache_spec.gen.go Cache
//go:generate testkit bench -o cachetest/cache_bench.gen.go Cache

// Cache exercises methods that don't return error: slice return,
// scalar return, pointer return, and void (no return at all).
type Cache interface {
	// Keys returns all keys. Slice return, no error.
	Keys(ctx context.Context) []string
	// Count returns the number of entries. Scalar return, no error.
	Count(ctx context.Context) int
	// Lookup returns a pointer — nil means not found. No error.
	Lookup(ctx context.Context, key string) *string
	// Clear removes all entries. Void — no return at all.
	Clear(ctx context.Context)
}
