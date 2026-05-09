// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bindings"
)

// PureContext provides a typed factory and call function to
// Pure-shape bench primitives.
type PureContext[T, R any] struct {
	B *testing.B
	bindings.PureBindings[T, R]
}

// Pure is a typed bench primitive for Pure-shaped methods.
type Pure[T, R any] func(PureContext[T, R])

// PureHotPath measures the single-goroutine pure-call latency and
// allocation rate. Reports ns/op and allocs/op.
func PureHotPath[T, R any]() Pure[T, R] {
	return func(ctx PureContext[T, R]) {
		impl := ctx.Factory()
		HotPath(ctx.B, "hot-path", func() {
			_ = ctx.Call(impl)
		})
	}
}

// PureAllocsWithin measures allocations per pure call and fails
// the benchmark if allocs exceed maxAllocs.
func PureAllocsWithin[T, R any](maxAllocs int) Pure[T, R] {
	return func(ctx PureContext[T, R]) {
		impl := ctx.Factory()
		AllocsWithin(ctx.B, fmt.Sprintf("allocs-within-%d", maxAllocs), maxAllocs, func() {
			_ = ctx.Call(impl)
		})
	}
}

// PureLatencyWithin enforces a per-call latency budget and fails
// the benchmark if the measured per-call latency exceeds maxLatency.
func PureLatencyWithin[T, R any](maxLatency time.Duration) Pure[T, R] {
	return func(ctx PureContext[T, R]) {
		impl := ctx.Factory()
		LatencyWithin(ctx.B, fmt.Sprintf("latency-within-%v", maxLatency), maxLatency, func() {
			_ = ctx.Call(impl)
		})
	}
}

// PureLatencyPercentilesWithin enforces per-percentile latency
// budgets (p50/p95/p99/...) and fails the benchmark if any
// percentile exceeds its budget. Reports the measured p50/p95/p99
// values via b.ReportMetric regardless of whether they're budgeted.
func PureLatencyPercentilesWithin[T, R any](budgets map[float64]time.Duration) Pure[T, R] {
	return func(ctx PureContext[T, R]) {
		impl := ctx.Factory()
		LatencyPercentilesWithin(ctx.B, "percentiles", budgets, func() {
			_ = ctx.Call(impl)
		})
	}
}

// PureConcurrentThroughput measures pure-call throughput under
// contention. Uses b.RunParallel for correct iteration scaling.
// Reports ns/op and allocs/op.
func PureConcurrentThroughput[T, R any](parallelism int) Pure[T, R] {
	return func(ctx PureContext[T, R]) {
		impl := ctx.Factory()
		ConcurrentThroughput(ctx.B, fmt.Sprintf("concurrent-%d", parallelism), parallelism, func() {
			_ = ctx.Call(impl)
		})
	}
}
