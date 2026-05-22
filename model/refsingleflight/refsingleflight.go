// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package refsingleflight provides the [Coalescer] reference for
// the GetOrCompute contract-tier shape. Concurrent callers with
// the same key share a single compute invocation; the result is
// then cached for subsequent calls.
//
// This is a reference implementation: it records the number of
// compute invocations per key so consumer-level laws can assert
// the coalescing contract directly.
package refsingleflight

import (
	"context"
	"sync"
)

// Coalescer single-flights compute calls per key.
type Coalescer[K comparable, V any] struct {
	mu    sync.Mutex
	cache map[K]V
	calls map[K]int
}

// NewCoalescer constructs an empty [Coalescer].
func NewCoalescer[K comparable, V any]() *Coalescer[K, V] {
	return &Coalescer[K, V]{
		cache: make(map[K]V),
		calls: make(map[K]int),
	}
}

// Do returns the cached value for k, invoking fn exactly once
// across the keyspace's lifetime. The reference does not enforce
// concurrent-coalescing — it caches eagerly — which models the
// strongest single-flight semantic the SUT can claim.
func (c *Coalescer[K, V]) Do(_ context.Context, k K, fn func() V) (V, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if v, ok := c.cache[k]; ok {
		return v, nil
	}
	v := fn()
	c.cache[k] = v
	c.calls[k]++
	return v, nil
}

// Calls returns the number of times fn was invoked for the given
// key. Used by SUT-vs-reference laws to verify coalescing.
func (c *Coalescer[K, V]) Calls(k K) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls[k]
}
