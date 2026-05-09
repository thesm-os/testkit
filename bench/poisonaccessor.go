// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bindings"
)

// PoisonAccessorContext provides a typed factory and call function to
// PoisonAccessor-shape bench primitives.
type PoisonAccessorContext[T any] struct {
	B *testing.B
	bindings.PoisonAccessorBindings[T]
}

// PoisonAccessor is a typed bench primitive for PoisonAccessor-shaped methods.
type PoisonAccessor[T any] func(PoisonAccessorContext[T])

// PoisonAccessorHotPath measures the single-goroutine accessor-call
// latency and allocation rate. Reports ns/op and allocs/op.
func PoisonAccessorHotPath[T any]() PoisonAccessor[T] {
	return func(ctx PoisonAccessorContext[T]) {
		impl := ctx.Factory()
		HotPath(ctx.B, "hot-path", func() {
			errSink = ctx.Call(impl)
		})
	}
}

// PoisonAccessorAllocsWithin measures allocations per accessor call
// and fails the benchmark if allocs exceed maxAllocs.
func PoisonAccessorAllocsWithin[T any](maxAllocs int) PoisonAccessor[T] {
	return func(ctx PoisonAccessorContext[T]) {
		impl := ctx.Factory()
		AllocsWithin(ctx.B, fmt.Sprintf("allocs-within-%d", maxAllocs), maxAllocs, func() {
			errSink = ctx.Call(impl)
		})
	}
}

// PoisonAccessorLatencyWithin enforces a per-call latency budget and
// fails the benchmark if the measured per-call latency exceeds
// maxLatency.
func PoisonAccessorLatencyWithin[T any](maxLatency time.Duration) PoisonAccessor[T] {
	return func(ctx PoisonAccessorContext[T]) {
		impl := ctx.Factory()
		LatencyWithin(ctx.B, fmt.Sprintf("latency-within-%v", maxLatency), maxLatency, func() {
			errSink = ctx.Call(impl)
		})
	}
}

// PoisonAccessorLatencyPercentilesWithin enforces per-percentile
// latency budgets (p50/p95/p99/...) and fails the benchmark if any
// percentile exceeds its budget. Reports the measured p50/p95/p99
// values via b.ReportMetric regardless of whether they're budgeted.
func PoisonAccessorLatencyPercentilesWithin[T any](budgets map[float64]time.Duration) PoisonAccessor[T] {
	return func(ctx PoisonAccessorContext[T]) {
		impl := ctx.Factory()
		LatencyPercentilesWithin(ctx.B, "percentiles", budgets, func() {
			errSink = ctx.Call(impl)
		})
	}
}

// PoisonAccessorConcurrentThroughput measures accessor-call throughput
// under contention. Uses b.RunParallel for correct iteration scaling.
func PoisonAccessorConcurrentThroughput[T any](parallelism int) PoisonAccessor[T] {
	return func(ctx PoisonAccessorContext[T]) {
		impl := ctx.Factory()
		ConcurrentThroughput(ctx.B, fmt.Sprintf("concurrent-%d", parallelism), parallelism, func() {
			errSink = ctx.Call(impl)
		})
	}
}
