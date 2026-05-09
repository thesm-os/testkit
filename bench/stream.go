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
type StreamContext[T, V any] struct {
	B *testing.B
	bindings.StreamBindings[T, V]
}

// Stream is a typed bench primitive for StreamReader-shaped methods.
type Stream[T, V any] func(StreamContext[T, V])

// StreamHotPath measures the full iteration latency and allocation rate.
// Reports ns/op and allocs/op (per full iteration of the stream).
func StreamHotPath[T, V any]() Stream[T, V] {
	return func(ctx StreamContext[T, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		HotPath(ctx.B, "iterate-all", func() {
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
		AllocsWithin(ctx.B, fmt.Sprintf("allocs-within-%d", maxAllocs), maxAllocs, func() {
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
		LatencyWithin(ctx.B, fmt.Sprintf("latency-within-%v", maxLatency), maxLatency, func() {
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
		ConcurrentThroughput(ctx.B, fmt.Sprintf("concurrent-%d", parallelism), parallelism, func() {
			for _, err := range ctx.Call(bctx, impl) {
				errSink = err
			}
		})
	}
}
