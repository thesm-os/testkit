// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bindings"
)

// PredicateContext provides a typed factory and call function to
// Predicate-shape bench primitives.
type PredicateContext[T any] struct {
	B *testing.B
	bindings.PredicateBindings[T]
}

// Predicate is a typed bench primitive for Predicate-shaped methods.
type Predicate[T any] func(PredicateContext[T])

// PredicateHotPath measures the single-goroutine predicate-call
// latency and allocation rate. Reports ns/op and allocs/op.
func PredicateHotPath[T any]() Predicate[T] {
	return func(ctx PredicateContext[T]) {
		impl := ctx.Factory()
		HotPath(ctx.B, "hot-path", func() {
			_ = ctx.Call(impl)
		})
	}
}

// PredicateAllocsWithin measures allocations per predicate call and fails
// the benchmark if allocs exceed maxAllocs.
func PredicateAllocsWithin[T any](maxAllocs int) Predicate[T] {
	return func(ctx PredicateContext[T]) {
		impl := ctx.Factory()
		AllocsWithin(ctx.B, fmt.Sprintf("allocs-within-%d", maxAllocs), maxAllocs, func() {
			_ = ctx.Call(impl)
		})
	}
}

// PredicateLatencyWithin enforces a per-call latency budget and fails
// the benchmark if the measured per-call latency exceeds maxLatency.
func PredicateLatencyWithin[T any](maxLatency time.Duration) Predicate[T] {
	return func(ctx PredicateContext[T]) {
		impl := ctx.Factory()
		LatencyWithin(ctx.B, fmt.Sprintf("latency-within-%v", maxLatency), maxLatency, func() {
			_ = ctx.Call(impl)
		})
	}
}

// PredicateLatencyPercentilesWithin enforces per-percentile latency
// budgets (p50/p95/p99/...) and fails the benchmark if any
// percentile exceeds its budget. Reports the measured p50/p95/p99
// values via b.ReportMetric regardless of whether they're budgeted.
func PredicateLatencyPercentilesWithin[T any](budgets map[float64]time.Duration) Predicate[T] {
	return func(ctx PredicateContext[T]) {
		impl := ctx.Factory()
		LatencyPercentilesWithin(ctx.B, "percentiles", budgets, func() {
			_ = ctx.Call(impl)
		})
	}
}

// PredicateConcurrentThroughput measures predicate-call throughput
// under contention. Uses b.RunParallel for correct iteration scaling.
func PredicateConcurrentThroughput[T any](parallelism int) Predicate[T] {
	return func(ctx PredicateContext[T]) {
		impl := ctx.Factory()
		ConcurrentThroughput(ctx.B, fmt.Sprintf("concurrent-%d", parallelism), parallelism, func() {
			_ = ctx.Call(impl)
		})
	}
}
