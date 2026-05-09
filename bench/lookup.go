// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bindings"
)

// LookupContext provides a typed factory and call function to
// Lookup-shape bench primitives.
type LookupContext[T any, K comparable, V, R any] struct {
	B *testing.B
	bindings.LookupBindings[T, K, V, R]
}

// Lookup is a typed bench primitive for Lookup-shaped methods.
type Lookup[T any, K comparable, V, R any] func(LookupContext[T, K, V, R])

// LookupHotPath measures lookup throughput against a known key.
func LookupHotPath[T any, K comparable, V, R any](key K) Lookup[T, K, V, R] {
	return func(ctx LookupContext[T, K, V, R]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		HotPath(ctx.B, "hot-path/"+SubtestKey(key), func() {
			_, _, _ = ctx.Call(bctx, impl, key)
		})
	}
}

// LookupAllocsWithin measures allocations per lookup and fails
// the benchmark if allocs exceed maxAllocs.
func LookupAllocsWithin[T any, K comparable, V, R any](key K, maxAllocs int) Lookup[T, K, V, R] {
	return func(ctx LookupContext[T, K, V, R]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		AllocsWithin(ctx.B, fmt.Sprintf("allocs-within-%d/%s", maxAllocs, SubtestKey(key)), maxAllocs, func() {
			_, _, _ = ctx.Call(bctx, impl, key)
		})
	}
}

// LookupLatencyWithin enforces a per-call latency budget and fails
// the benchmark if the measured per-call latency exceeds maxLatency.
func LookupLatencyWithin[T any, K comparable, V, R any](key K, maxLatency time.Duration) Lookup[T, K, V, R] {
	return func(ctx LookupContext[T, K, V, R]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		LatencyWithin(ctx.B, fmt.Sprintf("latency-within-%v/%s", maxLatency, SubtestKey(key)), maxLatency, func() {
			_, _, _ = ctx.Call(bctx, impl, key)
		})
	}
}

// LookupLatencyPercentilesWithin enforces per-percentile latency
// budgets (p50/p95/p99/...) and fails the benchmark if any
// percentile exceeds its budget. Reports the measured p50/p95/p99
// values via b.ReportMetric regardless of whether they're budgeted.
func LookupLatencyPercentilesWithin[T any, K comparable, V, R any](
	key K,
	budgets map[float64]time.Duration,
) Lookup[T, K, V, R] {
	return func(ctx LookupContext[T, K, V, R]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		LatencyPercentilesWithin(ctx.B, "percentiles/"+SubtestKey(key), budgets, func() {
			_, _, _ = ctx.Call(bctx, impl, key)
		})
	}
}

// LookupConcurrentThroughput measures lookup throughput under
// contention. Uses b.RunParallel for correct iteration scaling.
// Reports ns/op and allocs/op.
func LookupConcurrentThroughput[T any, K comparable, V, R any](key K, parallelism int) Lookup[T, K, V, R] {
	return func(ctx LookupContext[T, K, V, R]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		ConcurrentThroughput(ctx.B, fmt.Sprintf("concurrent-%d/%s", parallelism, SubtestKey(key)), parallelism, func() {
			_, _, _ = ctx.Call(bctx, impl, key)
		})
	}
}
