// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generics

//go:generate testkit builder -o genericstest/constraint.gen.go Stat

// Numeric is a custom constraint with a type-set union — the
// rarer-but-real shape that the builder must render verbatim
// (constraint name, not the underlying union) at the type-param
// position.
type Numeric interface{ ~int | ~int64 | ~float64 }

// Stat exercises the custom-constraint case. The builder picks
// `int` (the first concrete type satisfying the constraint by
// convention) for the test instantiation.
type Stat[T Numeric] struct {
	Value T
	Count int
}
