// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bindings"
)

// VoidLifecycleContext provides a typed factory and call function to
// VoidLifecycle-shape bench primitives. A VoidLifecycle-shaped method
// has the signature `func()` or `func(ctx)` — Reset, parameterless
// lifecycle hooks. The Call adapter accepts ctx uniformly; impls
// without ctx parameters ignore the argument.
type VoidLifecycleContext[T any] struct {
	B *testing.B
	bindings.VoidLifecycleBindings[T]
}

// VoidLifecycle is a typed bench primitive for VoidLifecycle-shaped methods.
type VoidLifecycle[T any] func(VoidLifecycleContext[T])

// VoidLifecycleHotPath measures the single-goroutine lifecycle-call
// latency and allocation rate. As with [LifecycleHotPath], repeated
// calls accumulate state in the impl; benchmarks should ensure the
// impl is idempotent under repeated invocation.
func VoidLifecycleHotPath[T any]() VoidLifecycle[T] {
	return func(ctx VoidLifecycleContext[T]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		HotPath(ctx.B, "hot-path", func() {
			ctx.Call(bctx, impl)
		})
	}
}

// VoidLifecycleAllocsWithin measures allocations per lifecycle call
// and fails the benchmark if allocs exceed maxAllocs.
func VoidLifecycleAllocsWithin[T any](maxAllocs int) VoidLifecycle[T] {
	return func(ctx VoidLifecycleContext[T]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		AllocsWithin(ctx.B, fmt.Sprintf("allocs-within-%d", maxAllocs), maxAllocs, func() {
			ctx.Call(bctx, impl)
		})
	}
}

// VoidLifecycleLatencyWithin enforces a per-call latency budget and
// fails the benchmark if the measured per-call latency exceeds
// maxLatency.
func VoidLifecycleLatencyWithin[T any](maxLatency time.Duration) VoidLifecycle[T] {
	return func(ctx VoidLifecycleContext[T]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		LatencyWithin(ctx.B, fmt.Sprintf("latency-within-%v", maxLatency), maxLatency, func() {
			ctx.Call(bctx, impl)
		})
	}
}

// VoidLifecycleLatencyPercentilesWithin enforces per-percentile
// latency budgets (p50/p95/p99/...) and fails the benchmark if any
// percentile exceeds its budget. Reports the measured p50/p95/p99
// values via b.ReportMetric regardless of whether they're budgeted.
func VoidLifecycleLatencyPercentilesWithin[T any](budgets map[float64]time.Duration) VoidLifecycle[T] {
	return func(ctx VoidLifecycleContext[T]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		LatencyPercentilesWithin(ctx.B, "percentiles", budgets, func() {
			ctx.Call(bctx, impl)
		})
	}
}

// VoidLifecycleConcurrentThroughput measures lifecycle-call throughput
// under contention. Uses b.RunParallel for correct iteration scaling.
// The impl must be safe for concurrent invocation.
func VoidLifecycleConcurrentThroughput[T any](parallelism int) VoidLifecycle[T] {
	return func(ctx VoidLifecycleContext[T]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		ConcurrentThroughput(ctx.B, fmt.Sprintf("concurrent-%d", parallelism), parallelism, func() {
			ctx.Call(bctx, impl)
		})
	}
}
