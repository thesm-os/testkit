// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package voidctx exercises void methods that take context + params
// but return nothing. These must NOT be classified as Writer (which
// requires an error return) — they should be Unknown.
package voidctx

import "context"

//go:generate testkit suite -o countertest/counter_spec.gen.go Counter
//go:generate testkit bench -o countertest/counter_bench.gen.go Counter

// Counter is a telemetry-style interface where mutation methods
// return nothing.
type Counter interface {
	// Add increments the counter. Void return — must be Unknown, not Writer.
	Add(ctx context.Context, value int64)

	// Name returns the counter name — Pure shape.
	Name() string
}
