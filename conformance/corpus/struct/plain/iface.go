// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package plain is the zero-value half of the builder's defaults handling: no
// Defaults companion, no field directives, nothing to seed from.
//
// It is the floor the other two fixtures are measured against. A generator
// that always emits a defaults-aware constructor produces identical output
// here and for the directive-bearing case, and only a fixture with no defaults
// at all can tell those apart.
package plain

// Item has no defaults of any kind, so New<Item>() must return a builder over
// the zero value.
//
//testkit:out plaintest/ pkg=plaintest
//testkit:builder
type Item struct {
	ID    string
	Count int
	Tags  []string
}
