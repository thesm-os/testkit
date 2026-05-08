// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generics

//go:generate testkit builder -o genericstest/single.gen.go Container

// Container is the single-type-parameter case with `any`
// constraint. The type parameter appears as a slice element — the
// builder must render `[]T` in source and `[]string` (concrete) in
// the generated test.
type Container[T any] struct {
	Label string
	Items []T
	Limit int
}
