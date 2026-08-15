// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import "pgregory.net/rapid"

// pair is a typed holder for two values used for composed multi-interface
// model testing. The consumer creates a factory that returns a linked
// pair where both components share state:
//
//	factory := func() model.pair[Store, Ledger] {
//	    ledger := NewLedger()
//	    store := NewStore(ledger) // linked: Put appends to ledger
//	    return model.pair[Store, Ledger]{A: store, B: ledger}
//	}
//
// Actions dispatch into pair fields via [liftA] and [liftB].
// Cross-interface laws are typed [law.Law[pair[A, B]]].
//
// pair does NOT compose interfaces via embedding — Go generics don't
// support embedding type parameters. Access is via typed fields:
// pair.A.StoreMethod(), pair.B.LedgerMethod().
type pair[A, B any] struct {
	A A
	B B
}

// triple is a typed holder for three values. Same pattern as [pair]
// for three-way composition.
type triple[A, B, C any] struct {
	A A
	B B
	C C
}

// liftA wraps an Action[A] into an Action[pair[A, B]] by routing
// through pair.A. Use this to reuse single-interface action helpers
// in composed tests.
func liftA[A, B any](a Action[A]) Action[pair[A, B]] {
	return Action[pair[A, B]]{
		Name: "A." + a.Name,
		Kind: a.Kind,
		Run: func(rt *rapid.T, sut, ref pair[A, B]) ActionResult {
			return a.Run(rt, sut.A, ref.A)
		},
	}
}

// liftB wraps an Action[B] into an Action[pair[A, B]] by routing
// through pair.B.
func liftB[A, B any](b Action[B]) Action[pair[A, B]] {
	return Action[pair[A, B]]{
		Name: "B." + b.Name,
		Kind: b.Kind,
		Run: func(rt *rapid.T, sut, ref pair[A, B]) ActionResult {
			return b.Run(rt, sut.B, ref.B)
		},
	}
}

// liftTripleA wraps an Action[A] into an Action[triple[A, B, C]].
func liftTripleA[A, B, C any](a Action[A]) Action[triple[A, B, C]] {
	return Action[triple[A, B, C]]{
		Name: "A." + a.Name,
		Kind: a.Kind,
		Run: func(rt *rapid.T, sut, ref triple[A, B, C]) ActionResult {
			return a.Run(rt, sut.A, ref.A)
		},
	}
}

// liftTripleB wraps an Action[B] into an Action[triple[A, B, C]].
func liftTripleB[A, B, C any](b Action[B]) Action[triple[A, B, C]] {
	return Action[triple[A, B, C]]{
		Name: "B." + b.Name,
		Kind: b.Kind,
		Run: func(rt *rapid.T, sut, ref triple[A, B, C]) ActionResult {
			return b.Run(rt, sut.B, ref.B)
		},
	}
}

// liftTripleC wraps an Action[C] into an Action[triple[A, B, C]].
func liftTripleC[A, B, C any](c Action[C]) Action[triple[A, B, C]] {
	return Action[triple[A, B, C]]{
		Name: "C." + c.Name,
		Kind: c.Kind,
		Run: func(rt *rapid.T, sut, ref triple[A, B, C]) ActionResult {
			return c.Run(rt, sut.C, ref.C)
		},
	}
}
