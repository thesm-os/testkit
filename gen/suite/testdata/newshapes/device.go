// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package newshapes exercises the four shapes that were previously
// missing typed plug-ins: Mutator, ReaderWithBool, Lookup, and
// PoisonAccessor.
package newshapes

import "context"

//go:generate testkit suite -o devicetest/device_spec.gen.go Device
//go:generate testkit bench -o devicetest/device_bench.gen.go Device

// Metadata holds device metadata returned by Lookup.
type Metadata struct {
	Firmware string
}

// Device exercises Mutator, ReaderWithBool, Lookup, and PoisonAccessor shapes.
type Device interface {
	// Increment is Mutator-shaped: func(ctx, V) — void return.
	//testkit:mutator
	Increment(ctx context.Context, delta int64)

	// Load is ReaderWithBool-shaped: func(ctx, K) (V, bool).
	Load(ctx context.Context, key string) (int64, bool)

	// Inspect is Lookup-shaped: func(K) (R1, R2, bool).
	Inspect(key string) (int64, Metadata, bool)

	// Err is PoisonAccessor-shaped: func() error.
	Err() error
}
