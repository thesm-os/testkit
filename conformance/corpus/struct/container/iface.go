// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package container is the generic half of the struct-kind corpus.
//
// Type parameters have to survive end to end: the builder type carries them,
// its constructor carries them, and every setter that touches a parameterised
// field takes the parameter rather than its constraint. A generator that
// erases them produces code which compiles only when T happens to be the
// constraint's underlying type, so the failure shows up for one instantiation
// and not another.
package container

// Container is generic over one unconstrained parameter, which is the case
// where nothing about T can be assumed.
type Container[T any] struct {
	Value T

	// Items is a slice of the parameter, so its Append takes T rather than any.
	Items []T

	// Label is not parameterised, so its setter must not acquire a type
	// parameter it does not need.
	Label string
}

// Pair is generic over two parameters with different constraints. A generator
// that assumes a single parameter, or that assumes every parameter is `any`,
// gets this wrong.
type Pair[K comparable, V any] struct {
	Key K

	Value V

	// Index is keyed by the comparable parameter, so the map setter has to
	// thread both parameters through.
	Index map[K]V
}
