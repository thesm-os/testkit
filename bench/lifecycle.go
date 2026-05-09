// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bindings"
)

// LifecycleContext provides a typed factory and call function to
// Lifecycle-shape bench primitives.
type LifecycleContext[T any] struct {
	B *testing.B
	bindings.LifecycleBindings[T]
}

// Lifecycle is a typed bench primitive for Lifecycle-shaped methods.
type Lifecycle[T any] func(LifecycleContext[T])

// LifecycleHotPath measures the single-goroutine lifecycle-call
// latency and allocation rate. Lifecycle methods are typically
// single-shot (Open/Close/Reset); the benchmark measures the cost of
// calling repeatedly against the same impl, so the impl should be
// idempotent under repeated invocation or the consumer should use a
// factory that reinitializes between primitive invocations.
func LifecycleHotPath[T any]() Lifecycle[T] {
	return func(ctx LifecycleContext[T]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		HotPath(ctx.B, "hot-path", func() {
			errSink = ctx.Call(bctx, impl)
		})
	}
}

// LifecycleAllocsWithin measures allocations per lifecycle call and fails
// the benchmark if allocs exceed maxAllocs.
func LifecycleAllocsWithin[T any](maxAllocs int) Lifecycle[T] {
	return func(ctx LifecycleContext[T]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		AllocsWithin(ctx.B, fmt.Sprintf("allocs-within-%d", maxAllocs), maxAllocs, func() {
			errSink = ctx.Call(bctx, impl)
		})
	}
}

// LifecycleLatencyWithin enforces a per-call latency budget and fails
// the benchmark if the measured per-call latency exceeds maxLatency.
func LifecycleLatencyWithin[T any](maxLatency time.Duration) Lifecycle[T] {
	return func(ctx LifecycleContext[T]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		LatencyWithin(ctx.B, fmt.Sprintf("latency-within-%v", maxLatency), maxLatency, func() {
			errSink = ctx.Call(bctx, impl)
		})
	}
}

// LifecycleConcurrentThroughput measures lifecycle-call throughput
// under contention. Uses b.RunParallel for correct iteration scaling.
// The impl must be safe for concurrent invocation (idempotent or
// internally serialized).
func LifecycleConcurrentThroughput[T any](parallelism int) Lifecycle[T] {
	return func(ctx LifecycleContext[T]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		ConcurrentThroughput(ctx.B, fmt.Sprintf("concurrent-%d", parallelism), parallelism, func() {
			errSink = ctx.Call(bctx, impl)
		})
	}
}
