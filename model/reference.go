// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import "pgregory.net/rapid"

// Pair is a typed holder for two values used for composed multi-interface
// model testing. The consumer creates a factory that returns a linked
// Pair where both components share state:
//
//	factory := func() model.Pair[Store, Ledger] {
//	    ledger := NewLedger()
//	    store := NewStore(ledger) // linked: Put appends to ledger
//	    return model.Pair[Store, Ledger]{A: store, B: ledger}
//	}
//
// Actions dispatch into Pair fields via [LiftA] and [LiftB].
// Cross-interface laws are typed [law.Law[Pair[A, B]]].
//
// Pair does NOT compose interfaces via embedding — Go generics don't
// support embedding type parameters. Access is via typed fields:
// pair.A.StoreMethod(), pair.B.LedgerMethod().
type Pair[A, B any] struct {
	A A
	B B
}

// Triple is a typed holder for three values. Same pattern as [Pair]
// for three-way composition.
type Triple[A, B, C any] struct {
	A A
	B B
	C C
}

// LiftA wraps an Action[A] into an Action[Pair[A, B]] by routing
// through pair.A. Use this to reuse single-interface action helpers
// in composed tests.
func LiftA[A, B any](a Action[A]) Action[Pair[A, B]] {
	return Action[Pair[A, B]]{
		Name: "A." + a.Name,
		Run: func(rt *rapid.T, sut, ref Pair[A, B]) {
			a.Run(rt, sut.A, ref.A)
		},
	}
}

// LiftB wraps an Action[B] into an Action[Pair[A, B]] by routing
// through pair.B.
func LiftB[A, B any](b Action[B]) Action[Pair[A, B]] {
	return Action[Pair[A, B]]{
		Name: "B." + b.Name,
		Run: func(rt *rapid.T, sut, ref Pair[A, B]) {
			b.Run(rt, sut.B, ref.B)
		},
	}
}

// LiftTripleA wraps an Action[A] into an Action[Triple[A, B, C]].
func LiftTripleA[A, B, C any](a Action[A]) Action[Triple[A, B, C]] {
	return Action[Triple[A, B, C]]{
		Name: "A." + a.Name,
		Run: func(rt *rapid.T, sut, ref Triple[A, B, C]) {
			a.Run(rt, sut.A, ref.A)
		},
	}
}

// LiftTripleB wraps an Action[B] into an Action[Triple[A, B, C]].
func LiftTripleB[A, B, C any](b Action[B]) Action[Triple[A, B, C]] {
	return Action[Triple[A, B, C]]{
		Name: "B." + b.Name,
		Run: func(rt *rapid.T, sut, ref Triple[A, B, C]) {
			b.Run(rt, sut.B, ref.B)
		},
	}
}

// LiftTripleC wraps an Action[C] into an Action[Triple[A, B, C]].
func LiftTripleC[A, B, C any](c Action[C]) Action[Triple[A, B, C]] {
	return Action[Triple[A, B, C]]{
		Name: "C." + c.Name,
		Run: func(rt *rapid.T, sut, ref Triple[A, B, C]) {
			c.Run(rt, sut.C, ref.C)
		},
	}
}
