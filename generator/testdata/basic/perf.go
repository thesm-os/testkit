// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

//go:generate testkit bench -o storetest/perf_bench.gen.go Perf

import "context"

// Perf exercises the bench generator's opt-in budget directives:
// `//testkit:allocs N`, `//testkit:latency D`, and
// `//testkit:percentiles pXX=D...`. Each method declares all three so
// the bench generator emits AllocsWithin, LatencyWithin, and
// LatencyPercentilesWithin gates alongside the always-emitted
// HotPath / ConcurrentThroughput primitives.
//
// Used by generator/bench tests to validate that directive presence
// drives the gate emission.
type Perf interface {
	// Hot is a Reader-shaped method with all three budget directives.
	//
	//testkit:allocs 0
	//testkit:latency 100us
	//testkit:percentiles p50=50us p95=200us p99=500us
	Hot(ctx context.Context, key string) (Item, error)
}
