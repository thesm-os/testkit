// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package multireturn exercises the spec generator with methods that return
// multiple non-error values. Tests pure observer type rendering and bounded
// result selection from multi-return methods.
package multireturn

import "context"

//go:generate testkit suite -o servicetest/service_spec.gen.go Service

// Stats holds service statistics.
type Stats struct {
	Active  int
	Pending int
}

// Service has methods with multiple non-error returns.
type Service interface {
	//testkit:pure
	// Status returns current stats and a health string.
	Status(ctx context.Context) (Stats, string, error)

	//testkit:nilsafe
	//testkit:ctx
	// Reset clears state and returns counts of cleared items.
	Reset(ctx context.Context) error
}
