// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package slice exercises the suite generator's type rendering
// against an interface whose value type is a slice ([]byte). This
// fixture lives in the generics-nested matrix to verify the
// generator emits valid Go for nested type patterns; the fixture
// itself is non-generic so the test instantiation is unambiguous
// (no per-position type-parameter substitution to worry about).
package slice

//go:generate testkit suite -o slicetest/cache_spec.gen_test.go ByteCache

import "context"

// ByteCache is a key→[]byte cache. The slice value type stresses
// the generator's []byte rendering — sample defaults, slice
// literals, idiomatic equality.
type ByteCache interface {
	// Reader-shape with V = []byte.
	Get(ctx context.Context, key string) ([]byte, error)
}
