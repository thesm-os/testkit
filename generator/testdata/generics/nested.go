// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generics

//go:generate testkit builder -o genericstest/nested.gen.go Cache

// Cache exercises type parameters appearing in every position the
// builder generator must handle: K as a map key (constrained to
// comparable so the position is type-system-valid), V as a map
// value, V as a slice element, and V as a scalar field. One type
// fixture covers four substitution paths.
type Cache[K comparable, V any] struct {
	Entries map[K]V
	Recent  []V
	Default V
}
