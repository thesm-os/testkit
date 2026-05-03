// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

// Pair is a typed holder for two values. It does NOT compose
// interfaces via embedding — Go generics don't support embedding
// type parameters. Consumers hand-write composed types when they
// need interface promotion. The generator emits concrete composed
// structs for each wired composition.
//
// Pair provides type-safe access to both components:
//
//	p := model.Pair[*StoreRef, *LedgerRef]{A: storeRef, B: ledgerRef}
//	p.A.Get(ctx, key)
//	p.B.Append(ctx, entry)
type Pair[A, B any] struct {
	A A
	B B
}

// Triple is a typed holder for three values.
type Triple[A, B, C any] struct {
	A A
	B B
	C C
}
