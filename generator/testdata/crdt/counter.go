// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package crdt is the testdata fixture for the //testkit:crdt-merge
// directive. The interface declares a state-converging counter
// whose Merge method commutes (Merge(a); Merge(b) and Merge(b);
// Merge(a) yield the same value) and is idempotent on its own
// state.
//
// The suite generator emits an AssertCRDTMerge subtest for any
// method declaring //testkit:crdt-merge Other=<peer>; the contract
// builds two impls, applies the same operations in opposite orders,
// and asserts state equality. Additive merge satisfies the CRDT
// laws so the contract passes against the additiveCounter in-mem.
package crdt

//go:generate testkit suite -o crdttest/counter_spec.gen_test.go AdditiveCounter

import "context"

// AdditiveCounter is a one-method CRDT — Merge folds an int delta
// into the local state. Apply (Merge(a), Merge(b)) on one impl and
// (Merge(b), Merge(a)) on another and the resulting states are
// equal. Value reads the converged total.
type AdditiveCounter interface {
	// Merge folds n into local state. The CRDT-merge contract uses
	// the directive's Other arg to name the paired merge method —
	// this fixture's merge is symmetric (Merge is its own
	// counterpart), so Other=Merge.
	//
	//testkit:crdt-merge Merge
	Merge(ctx context.Context, n int) error

	// Value reads the converged total. Used to confirm two impls
	// reach equal state after opposite-order merges.
	Value(ctx context.Context) (int, error)
}
