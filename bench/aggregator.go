// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bindings"
)

// AggregatorContext provides a typed factory and call function to
// Aggregator-shape bench primitives.
type AggregatorContext[T, R any] struct {
	B *testing.B
	bindings.AggregatorBindings[T, R]
}

// Aggregator is a typed bench primitive for Aggregator-shaped methods.
type Aggregator[T, R any] func(AggregatorContext[T, R])

// AggregatorHotPath measures the single-goroutine aggregation latency
// and allocation rate. Reports ns/op and allocs/op.
func AggregatorHotPath[T, R any]() Aggregator[T, R] {
	return func(ctx AggregatorContext[T, R]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		HotPath(ctx.B, "hot-path", func() {
			_, errSink = ctx.Call(bctx, impl)
		})
	}
}

// AggregatorAllocsWithin measures allocations per aggregation and fails
// the benchmark if allocs exceed maxAllocs.
func AggregatorAllocsWithin[T, R any](maxAllocs int) Aggregator[T, R] {
	return func(ctx AggregatorContext[T, R]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		AllocsWithin(ctx.B, fmt.Sprintf("allocs-within-%d", maxAllocs), maxAllocs, func() {
			_, errSink = ctx.Call(bctx, impl)
		})
	}
}

// AggregatorLatencyWithin enforces a per-call latency budget and
// fails the benchmark if the measured per-call latency exceeds
// maxLatency.
func AggregatorLatencyWithin[T, R any](maxLatency time.Duration) Aggregator[T, R] {
	return func(ctx AggregatorContext[T, R]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		LatencyWithin(ctx.B, fmt.Sprintf("latency-within-%v", maxLatency), maxLatency, func() {
			_, errSink = ctx.Call(bctx, impl)
		})
	}
}

// AggregatorConcurrentThroughput measures aggregation throughput
// under contention. Uses b.RunParallel for correct iteration scaling.
func AggregatorConcurrentThroughput[T, R any](parallelism int) Aggregator[T, R] {
	return func(ctx AggregatorContext[T, R]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		ConcurrentThroughput(ctx.B, fmt.Sprintf("concurrent-%d", parallelism), parallelism, func() {
			_, errSink = ctx.Call(bctx, impl)
		})
	}
}
