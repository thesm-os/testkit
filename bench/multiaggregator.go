// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bindings"
)

// MultiAggregatorContext provides a typed factory and call function
// to MultiAggregator-shape bench primitives. A MultiAggregator-shaped
// method has the signature `func(ctx?) (V1, V2, error)` — 2-tuple
// aggregations like Stats(ctx) (count, total, error).
type MultiAggregatorContext[T any, V1, V2 any] struct {
	B *testing.B
	bindings.MultiAggregatorBindings[T, V1, V2]
}

// MultiAggregator is a typed bench primitive for MultiAggregator-shaped methods.
type MultiAggregator[T any, V1, V2 any] func(MultiAggregatorContext[T, V1, V2])

// MultiAggregatorHotPath measures the single-goroutine aggregation
// latency and allocation rate. Reports ns/op and allocs/op.
func MultiAggregatorHotPath[T, V1, V2 any]() MultiAggregator[T, V1, V2] {
	return func(ctx MultiAggregatorContext[T, V1, V2]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		HotPath(ctx.B, "hot-path", func() {
			_, _, errSink = ctx.Call(bctx, impl)
		})
	}
}

// MultiAggregatorAllocsWithin measures allocations per aggregation and
// fails the benchmark if allocs exceed maxAllocs.
func MultiAggregatorAllocsWithin[T, V1, V2 any](maxAllocs int) MultiAggregator[T, V1, V2] {
	return func(ctx MultiAggregatorContext[T, V1, V2]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		AllocsWithin(ctx.B, fmt.Sprintf("allocs-within-%d", maxAllocs), maxAllocs, func() {
			_, _, errSink = ctx.Call(bctx, impl)
		})
	}
}

// MultiAggregatorLatencyWithin enforces a per-call latency budget and
// fails the benchmark if the measured per-call latency exceeds
// maxLatency.
func MultiAggregatorLatencyWithin[T, V1, V2 any](maxLatency time.Duration) MultiAggregator[T, V1, V2] {
	return func(ctx MultiAggregatorContext[T, V1, V2]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		LatencyWithin(ctx.B, fmt.Sprintf("latency-within-%v", maxLatency), maxLatency, func() {
			_, _, errSink = ctx.Call(bctx, impl)
		})
	}
}

// MultiAggregatorConcurrentThroughput measures aggregation throughput
// under contention. Uses b.RunParallel for correct iteration scaling.
func MultiAggregatorConcurrentThroughput[T, V1, V2 any](parallelism int) MultiAggregator[T, V1, V2] {
	return func(ctx MultiAggregatorContext[T, V1, V2]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		ConcurrentThroughput(ctx.B, fmt.Sprintf("concurrent-%d", parallelism), parallelism, func() {
			_, _, errSink = ctx.Call(bctx, impl)
		})
	}
}
