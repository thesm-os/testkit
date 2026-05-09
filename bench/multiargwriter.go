// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bindings"
)

// MultiArgWriterContext provides a typed factory and call function to
// MultiArgWriter-shape bench primitives. A MultiArgWriter-shaped
// method has the signature `func(ctx, p1, p2, p3) error` — Watcher,
// Subscribe, Schedule patterns.
type MultiArgWriterContext[T any, P1, P2, P3 any] struct {
	B *testing.B
	bindings.MultiArgWriterBindings[T, P1, P2, P3]
}

// MultiArgWriter is a typed bench primitive for MultiArgWriter-shaped methods.
type MultiArgWriter[T any, P1, P2, P3 any] func(MultiArgWriterContext[T, P1, P2, P3])

// MultiArgWriterHotPath measures the single-goroutine multi-arg-write
// latency and allocation rate. Repeated calls accumulate state in the
// impl; benchmarks should pick params whose write is idempotent or use
// a factory that reinitializes between primitive invocations.
func MultiArgWriterHotPath[T, P1, P2, P3 any](p1 P1, p2 P2, p3 P3) MultiArgWriter[T, P1, P2, P3] {
	return func(ctx MultiArgWriterContext[T, P1, P2, P3]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		HotPath(ctx.B, "hot-path", func() {
			errSink = ctx.Call(bctx, impl, p1, p2, p3)
		})
	}
}

// MultiArgWriterAllocsWithin measures allocations per multi-arg write
// and fails the benchmark if allocs exceed maxAllocs.
func MultiArgWriterAllocsWithin[T, P1, P2, P3 any](p1 P1, p2 P2, p3 P3, maxAllocs int) MultiArgWriter[T, P1, P2, P3] {
	return func(ctx MultiArgWriterContext[T, P1, P2, P3]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		AllocsWithin(ctx.B, fmt.Sprintf("allocs-within-%d", maxAllocs), maxAllocs, func() {
			errSink = ctx.Call(bctx, impl, p1, p2, p3)
		})
	}
}

// MultiArgWriterLatencyWithin enforces a per-call latency budget and
// fails the benchmark if the measured per-call latency exceeds
// maxLatency.
func MultiArgWriterLatencyWithin[T, P1, P2, P3 any](
	p1 P1, p2 P2, p3 P3, maxLatency time.Duration,
) MultiArgWriter[T, P1, P2, P3] {
	return func(ctx MultiArgWriterContext[T, P1, P2, P3]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		name := fmt.Sprintf("latency-within-%v", maxLatency)
		LatencyWithin(ctx.B, name, maxLatency, func() {
			errSink = ctx.Call(bctx, impl, p1, p2, p3)
		})
	}
}

// MultiArgWriterConcurrentThroughput measures multi-arg-write throughput
// under contention. Uses b.RunParallel for correct iteration scaling.
func MultiArgWriterConcurrentThroughput[T, P1, P2, P3 any](
	p1 P1, p2 P2, p3 P3, parallelism int,
) MultiArgWriter[T, P1, P2, P3] {
	return func(ctx MultiArgWriterContext[T, P1, P2, P3]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		name := fmt.Sprintf("concurrent-%d", parallelism)
		ConcurrentThroughput(ctx.B, name, parallelism, func() {
			errSink = ctx.Call(bctx, impl, p1, p2, p3)
		})
	}
}
