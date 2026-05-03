// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package noncrud exercises the non-CRUD fallback path where refmap
// synthesis is unavailable (Lifecycle-only interface).
package noncrud

import "context"

//go:generate testkit model -o closertest/closer_model.gen.go Closer

// Closer is a Lifecycle-only interface — no Reader, no Writer.
// Tier 0 reference synthesis is unavailable; consumer must supply
// a reference via CloserModelReference.
type Closer interface {
	// Close is Lifecycle-shaped.
	Close(ctx context.Context) error

	// Ping is Lifecycle-shaped.
	Ping(ctx context.Context) error
}
