// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

//go:generate testkit bench -o storetest/perf_bench.gen.go Perf

import "context"

// Perf exercises the bench generator's opt-in budget directives:
// `//testkit:allocs N` and `//testkit:latency D`. Each method
// declares both directives so the bench generator emits an
// AllocsWithin gate AND a LatencyWithin gate alongside the
// always-emitted HotPath / ConcurrentThroughput primitives.
//
// Used by generator/bench tests to validate that the directive
// presence drives the gate emission.
type Perf interface {
	// Hot is a Reader-shaped method with both budget directives.
	//
	//testkit:allocs 0
	//testkit:latency 100us
	Hot(ctx context.Context, key string) (Item, error)
}
