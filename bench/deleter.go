// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bindings"
)

// DeleterContext provides a typed factory and call function to
// Deleter-shape bench primitives.
type DeleterContext[T any, K comparable] struct {
	B *testing.B
	bindings.DeleterBindings[T, K]
}

// Deleter is a typed bench primitive for Deleter-shaped methods.
type Deleter[T any, K comparable] func(DeleterContext[T, K])

// DeleterHotPath measures the single-goroutine delete latency and
// allocation rate for the given key. Reports ns/op and allocs/op.
// After the first iteration the key is gone — subsequent iterations
// exercise the not-found path. Use a factory that reinitializes
// between primitive invocations to control which path dominates.
func DeleterHotPath[T any, K comparable](key K) Deleter[T, K] {
	return func(ctx DeleterContext[T, K]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		HotPath(ctx.B, "hot-path/"+SubtestKey(key), func() {
			errSink = ctx.Call(bctx, impl, key)
		})
	}
}

// DeleterAllocsWithin measures allocations per delete and fails the
// benchmark if allocs exceed maxAllocs.
func DeleterAllocsWithin[T any, K comparable](key K, maxAllocs int) Deleter[T, K] {
	return func(ctx DeleterContext[T, K]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		AllocsWithin(ctx.B, fmt.Sprintf("allocs-within-%d/%s", maxAllocs, SubtestKey(key)), maxAllocs, func() {
			errSink = ctx.Call(bctx, impl, key)
		})
	}
}

// DeleterLatencyWithin enforces a per-call latency budget and fails
// the benchmark if the measured per-call latency exceeds maxLatency.
func DeleterLatencyWithin[T any, K comparable](key K, maxLatency time.Duration) Deleter[T, K] {
	return func(ctx DeleterContext[T, K]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		LatencyWithin(ctx.B, fmt.Sprintf("latency-within-%v/%s", maxLatency, SubtestKey(key)), maxLatency, func() {
			errSink = ctx.Call(bctx, impl, key)
		})
	}
}

// DeleterLatencyPercentilesWithin enforces per-percentile latency
// budgets (p50/p95/p99/...) and fails the benchmark if any
// percentile exceeds its budget. Reports the measured p50/p95/p99
// values via b.ReportMetric regardless of whether they're budgeted.
func DeleterLatencyPercentilesWithin[T any, K comparable](key K, budgets map[float64]time.Duration) Deleter[T, K] {
	return func(ctx DeleterContext[T, K]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		LatencyPercentilesWithin(ctx.B, "percentiles/"+SubtestKey(key), budgets, func() {
			errSink = ctx.Call(bctx, impl, key)
		})
	}
}

// DeleterConcurrentThroughput measures delete throughput under
// contention. Uses b.RunParallel for correct iteration scaling.
// Reports ns/op and allocs/op — typically dominated by the impl's
// synchronization strategy.
func DeleterConcurrentThroughput[T any, K comparable](key K, parallelism int) Deleter[T, K] {
	return func(ctx DeleterContext[T, K]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		ConcurrentThroughput(ctx.B, fmt.Sprintf("concurrent-%d/%s", parallelism, SubtestKey(key)), parallelism, func() {
			errSink = ctx.Call(bctx, impl, key)
		})
	}
}
