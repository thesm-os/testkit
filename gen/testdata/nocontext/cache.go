// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package nocontext exercises the spec generator with methods that have
// no context.Context parameter. The nilsafe directive works; ctx and timeout
// are not applicable.
package nocontext

import "errors"

//go:generate testkit suite    -o cachetest/cache_spec.gen.go  Cache
//go:generate testkit bench    -o cachetest/cache_bench.gen.go Cache
//go:generate testkit stub     -o cachetest/cache_stub.gen.go  Cache
//go:generate testkit sentinel -o errors.gen_test.go

// ErrMiss is returned when a key is not found.
var ErrMiss = errors.New("nocontext: cache miss")

// Cache is a simple cache without context parameters.
type Cache interface {
	//testkit:errors ErrMiss
	//testkit:nilsafe
	// Get returns a cached value.
	Get(key string) (string, error)

	//testkit:nilsafe
	// Set stores a value.
	Set(key string, value string) error

	//testkit:bounded 0 10000
	// Len returns the number of cached entries.
	Len() int
}
