// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generics

//go:generate testkit builder -o genericstest/builders.gen.go Container Pair

// Container holds items of a single type.
type Container[T any] struct {
	Label string
	Items []T
	Limit int
}

// Pair holds two values of different types.
type Pair[A, B any] struct {
	First  A
	Second B
}
