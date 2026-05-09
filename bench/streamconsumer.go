// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bindings"
)

// StreamConsumerContext provides a typed factory and call function to
// StreamConsumer-shape bench primitives. A StreamConsumer-shaped
// method has the signature `func(ctx, S) (V, error)` where S is
// interface-typed.
//
// BytesPerOp records bytes consumed per call (e.g. payload size of
// the stream input). When non-zero, primitives call b.SetBytes so
// `go test -bench` reports MB/s alongside ns/op.
type StreamConsumerContext[T, S, V any] struct {
	B *testing.B
	bindings.StreamConsumerBindings[T, S, V]
	BytesPerOp int64
}

// StreamConsumer is a typed bench primitive for StreamConsumer-shaped methods.
type StreamConsumer[T, S, V any] func(StreamConsumerContext[T, S, V])

// StreamConsumerHotPath measures the single-goroutine consume latency
// and allocation rate for the given stream input. The stream argument
// is passed by value across iterations; a stateful stream (e.g. one
// that reads from a connection) should be reset by the consumer or
// produced fresh inside `streamFactory` per iteration.
func StreamConsumerHotPath[T, S, V any](streamFactory func() S) StreamConsumer[T, S, V] {
	return func(ctx StreamConsumerContext[T, S, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		HotPathWithBytes(ctx.B, "hot-path", ctx.BytesPerOp, func() {
			_, errSink = ctx.Call(bctx, impl, streamFactory())
		})
	}
}

// StreamConsumerAllocsWithin measures allocations per consume call
// and fails the benchmark if allocs exceed maxAllocs.
func StreamConsumerAllocsWithin[T, S, V any](streamFactory func() S, maxAllocs int) StreamConsumer[T, S, V] {
	return func(ctx StreamConsumerContext[T, S, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		AllocsWithinWithBytes(ctx.B, fmt.Sprintf("allocs-within-%d", maxAllocs), ctx.BytesPerOp, maxAllocs, func() {
			_, errSink = ctx.Call(bctx, impl, streamFactory())
		})
	}
}

// StreamConsumerLatencyWithin enforces a per-call latency budget and
// fails the benchmark if the measured per-call latency exceeds
// maxLatency.
func StreamConsumerLatencyWithin[T, S, V any](
	streamFactory func() S, maxLatency time.Duration,
) StreamConsumer[T, S, V] {
	return func(ctx StreamConsumerContext[T, S, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		name := fmt.Sprintf("latency-within-%v", maxLatency)
		LatencyWithinWithBytes(ctx.B, name, ctx.BytesPerOp, maxLatency, func() {
			_, errSink = ctx.Call(bctx, impl, streamFactory())
		})
	}
}

// StreamConsumerConcurrentThroughput measures consume-call throughput
// under contention. Uses b.RunParallel for correct iteration scaling.
// `streamFactory` is invoked once per iteration on each goroutine —
// it must be safe for concurrent invocation.
func StreamConsumerConcurrentThroughput[T, S, V any](streamFactory func() S, parallelism int) StreamConsumer[T, S, V] {
	return func(ctx StreamConsumerContext[T, S, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		name := fmt.Sprintf("concurrent-%d", parallelism)
		ConcurrentThroughputWithBytes(ctx.B, name, ctx.BytesPerOp, parallelism, func() {
			_, errSink = ctx.Call(bctx, impl, streamFactory())
		})
	}
}
