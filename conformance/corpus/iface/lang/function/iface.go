// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package function is the language-axis fixture for free functions.
//
// eidos annotates package-level functions through the same three axes it
// applies to interface methods — its walker has an OnFunction hook alongside
// OnMethod. Every other fixture in this corpus declares an interface, so a
// generator or a gate that reads only methods would look complete against all
// of them and still miss every classification a consumer put on a plain
// function.
//
// The functions here carry a body rather than being declarations, because a
// free function has nowhere else to live: an interface method is a signature
// and a function is not.
package function

import (
	"context"
	"errors"
)

// ErrNotFound is the sentinel the reading function reports.
var ErrNotFound = errors.New("function: not found")

// Value is the payload these functions carry.
type Value struct{ Key, Body string }

// Get is a reader by signature alone, classified with no directive — the same
// inference that applies to a method of the same shape.
func Get(_ context.Context, key string) (Value, error) {
	if key == "" {
		return Value{}, ErrNotFound
	}
	return Value{Key: key}, nil
}

// Put carries a mixin, proving the directive axes reach a function and not
// only a method.
//
//testkit:mixin idempotent
func Put(_ context.Context, _ Value) error { return nil }

// Acquire hosts a contract from a free function, with Release as its partner.
// Contracts resolve partners by name within a scope, so a contract declared
// across two functions exercises a resolution path a method pair does not.
//
//testkit:contract lease role=acquire release=Release
func Acquire(_ context.Context, _ string) error { return nil }

// Release is the lease contract's release role.
func Release(_ context.Context, _ string) error { return nil }
