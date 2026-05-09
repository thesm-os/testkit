// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bindings"
)

// WriterContext provides a typed factory and call function to
// Writer-shape bench primitives.
type WriterContext[T, V any] struct {
	B *testing.B
	bindings.WriterBindings[T, V]
}

// Writer is a typed bench primitive for Writer-shaped methods.
// Primitives are either measurements (HotPath, ConcurrentThroughput)
// or gates (AllocsWithin, LatencyWithin).
type Writer[T, V any] func(WriterContext[T, V])

// WriterHotPath measures the single-goroutine write latency and
// allocation rate for the given value. Reports ns/op and allocs/op.
// Repeated writes accumulate in the impl; benchmarks should pick a
// value whose write is idempotent (e.g. overwriting the same key) or
// use a factory that reinitializes between primitive invocations.
func WriterHotPath[T, V any](sample V) Writer[T, V] {
	return func(ctx WriterContext[T, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		HotPath(ctx.B, "hot-path/"+SubtestKey(sample), func() {
			errSink = ctx.Call(bctx, impl, sample)
		})
	}
}

// WriterAllocsWithin measures allocations per write and fails the
// benchmark if allocs exceed maxAllocs. Deterministic gate suitable for CI.
func WriterAllocsWithin[T, V any](sample V, maxAllocs int) Writer[T, V] {
	return func(ctx WriterContext[T, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		AllocsWithin(ctx.B, fmt.Sprintf("allocs-within-%d/%s", maxAllocs, SubtestKey(sample)), maxAllocs, func() {
			errSink = ctx.Call(bctx, impl, sample)
		})
	}
}

// WriterLatencyWithin enforces a per-call latency budget and fails
// the benchmark if the measured per-call latency exceeds maxLatency.
// Deterministic gate suitable for CI.
func WriterLatencyWithin[T, V any](sample V, maxLatency time.Duration) Writer[T, V] {
	return func(ctx WriterContext[T, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		LatencyWithin(ctx.B, fmt.Sprintf("latency-within-%v/%s", maxLatency, SubtestKey(sample)), maxLatency, func() {
			errSink = ctx.Call(bctx, impl, sample)
		})
	}
}

// WriterConcurrentThroughput measures write throughput under contention.
// Uses b.RunParallel for correct iteration scaling. Reports ns/op and
// allocs/op — typically dominated by the impl's synchronization
// strategy (mutex, sync.Map, lock-striping, etc.).
func WriterConcurrentThroughput[T, V any](sample V, parallelism int) Writer[T, V] {
	return func(ctx WriterContext[T, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		name := fmt.Sprintf("concurrent-%d/%s", parallelism, SubtestKey(sample))
		ConcurrentThroughput(ctx.B, name, parallelism, func() {
			errSink = ctx.Call(bctx, impl, sample)
		})
	}
}
