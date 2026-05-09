// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bindings"
)

// StreamContext provides a typed factory and call function to
// Stream-shape bench primitives.
//
// BytesPerOp records the number of bytes processed per full
// iteration. When non-zero, every primitive that creates a b.Run
// scope calls b.SetBytes(BytesPerOp) so `go test -bench` reports
// MB/s alongside ns/op. Set this for streams of known per-element
// size (e.g. fixed-record protocols) so consumers comparing impls
// see throughput numbers, not just latency.
type StreamContext[T, V any] struct {
	B *testing.B
	bindings.StreamBindings[T, V]
	BytesPerOp int64
}

// Stream is a typed bench primitive for StreamReader-shaped methods.
type Stream[T, V any] func(StreamContext[T, V])

// StreamHotPath measures the full iteration latency and allocation rate.
// Reports ns/op and allocs/op (per full iteration of the stream).
func StreamHotPath[T, V any]() Stream[T, V] {
	return func(ctx StreamContext[T, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		HotPathWithBytes(ctx.B, "hot-path", ctx.BytesPerOp, func() {
			for _, err := range ctx.Call(bctx, impl) {
				errSink = err
			}
		})
	}
}

// StreamAllocsWithin measures allocations per full iteration and fails
// the benchmark if allocs exceed maxAllocs.
func StreamAllocsWithin[T, V any](maxAllocs int) Stream[T, V] {
	return func(ctx StreamContext[T, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		AllocsWithinWithBytes(ctx.B, fmt.Sprintf("allocs-within-%d", maxAllocs), ctx.BytesPerOp, maxAllocs, func() {
			for _, err := range ctx.Call(bctx, impl) {
				errSink = err
			}
		})
	}
}

// StreamLatencyWithin enforces a per-iteration latency budget and
// fails the benchmark if the measured per-iteration latency exceeds
// maxLatency. Per-iteration here means a full pass through the stream,
// not per-element.
func StreamLatencyWithin[T, V any](maxLatency time.Duration) Stream[T, V] {
	return func(ctx StreamContext[T, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		LatencyWithinWithBytes(ctx.B, fmt.Sprintf("latency-within-%v", maxLatency), ctx.BytesPerOp, maxLatency, func() {
			for _, err := range ctx.Call(bctx, impl) {
				errSink = err
			}
		})
	}
}

// StreamLatencyPercentilesWithin enforces per-percentile latency
// budgets (p50/p95/p99/...) for full stream iteration and fails the
// benchmark if any percentile exceeds its budget. Reports the measured
// p50/p95/p99 values via b.ReportMetric regardless of whether they're
// budgeted.
func StreamLatencyPercentilesWithin[T, V any](budgets map[float64]time.Duration) Stream[T, V] {
	return func(ctx StreamContext[T, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		LatencyPercentilesWithin(ctx.B, "percentiles", budgets, func() {
			for _, err := range ctx.Call(bctx, impl) {
				errSink = err
			}
		})
	}
}

// StreamConcurrentThroughput measures stream-iteration throughput
// under contention. Uses b.RunParallel for correct iteration scaling.
// Each goroutine independently materializes a fresh iter.Seq2 via
// ctx.Call and iterates to completion.
func StreamConcurrentThroughput[T, V any](parallelism int) Stream[T, V] {
	return func(ctx StreamContext[T, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		name := fmt.Sprintf("concurrent-%d", parallelism)
		ConcurrentThroughputWithBytes(ctx.B, name, ctx.BytesPerOp, parallelism, func() {
			for _, err := range ctx.Call(bctx, impl) {
				errSink = err
			}
		})
	}
}
