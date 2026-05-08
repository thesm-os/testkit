// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generics

//go:generate testkit builder -o genericstest/double.gen.go Pair

// Pair is the two-type-parameter case, both `any`-constrained.
// Both parameters appear as scalar fields — the builder must
// render `[A, B any]` in the type-param decl, `[A, B]` in the
// args, and `[string, int]` (concrete) for the test.
type Pair[A, B any] struct {
	First  A
	Second B
}
