// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package pointer exercises the suite generator's type rendering
// against an interface returning a pointer to a generic type
// (*Container[int]). Stresses pointer-of-generic value rendering.
package pointer

//go:generate testkit suite -o pointertest/cache_spec.gen_test.go ContainerCache

import "context"

// Container is a generic value-bearing struct used inside the
// pointer-cache value type. Its single field stores any T.
type Container[T any] struct {
	Value T
}

// ContainerCache is a key→*Container[int] cache.
type ContainerCache interface {
	// Reader-shape with V = *Container[int].
	Get(ctx context.Context, key string) (*Container[int], error)
}
