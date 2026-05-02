// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generics

// Cache is a generic key-value cache.
type Cache[K comparable, V any] interface {
	Get(key K) (V, bool)
	Set(key K, value V)
}

// Pair holds two values of different types.
type Pair[A, B any] struct {
	First  A
	Second B
}
