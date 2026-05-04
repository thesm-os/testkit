// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package newshapes exercises the Phase 1.5 shape extensions:
// ReaderWithBool, Mutator, and PoisonAccessor. Mirrors a simplified
// version of thesmos's kernel Machine interface.
package newshapes

import "context"

//go:generate testkit model -o machinetest/machine_model.gen.go Machine

// Command is a state-mutating input.
type Command struct {
	ID    string
	Value int
}

// State is a read-only snapshot.
type State struct {
	Total int
	Count int
}

// Machine exercises all three new shapes.
type Machine interface {
	// Fold is Mutator-shaped: func(ctx, V) with no return.
	// State changes are observed via State() and Err().
	//testkit:mutator
	Fold(ctx context.Context, cmd Command)

	// Lookup is ReaderWithBool-shaped: func(ctx, K) (V, bool).
	// Returns the command's accumulated value, or false if not seen.
	Lookup(ctx context.Context, id string) (State, bool)

	// State is Pure-shaped: func() T.
	State() State

	// Err is PoisonAccessor-shaped: func() error.
	Err() error
}
