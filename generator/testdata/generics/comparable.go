// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generics

//go:generate testkit builder -o genericstest/comparable.gen.go Lookup

// Lookup exercises the `comparable` built-in constraint, which the
// builder must render verbatim (not collapse to `any`). The
// comparable-constrained K is used as a map key — the position
// where the constraint is load-bearing.
type Lookup[K comparable, V any] struct {
	Entries map[K]V
	Recent  []V
}
