// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bindings"
)

// ReaderContext provides a typed factory and call function to
// Reader-shape bench primitives. Populated by generator-emitted
// BenchOn<Method> dispatch.
type ReaderContext[T any, K comparable, V any] struct {
	B *testing.B
	bindings.ReaderBindings[T, K, V]
}

// Reader is a typed bench primitive for Reader-shaped methods.
// Primitives are either measurements (HotPath, ConcurrentThroughput)
// or gates (AllocsWithin, LatencyWithin).
type Reader[T any, K comparable, V any] func(ReaderContext[T, K, V])

// ReaderHotPath measures the single-goroutine read latency and
// allocation rate for the given key. Reports ns/op and allocs/op.
// Measurement only — does not fail the benchmark.
func ReaderHotPath[T any, K comparable, V any](key K) Reader[T, K, V] {
	return func(ctx ReaderContext[T, K, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		HotPath(ctx.B, fmt.Sprintf("hot-path/%v", key), func() {
			_, errSink = ctx.Call(bctx, impl, key)
		})
	}
}

// ReaderAllocsWithin measures allocations per read and fails the
// benchmark if allocs exceed maxAllocs. Deterministic gate suitable for CI.
func ReaderAllocsWithin[T any, K comparable, V any](key K, maxAllocs int) Reader[T, K, V] {
	return func(ctx ReaderContext[T, K, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		AllocsWithin(ctx.B, fmt.Sprintf("allocs-within-%d/%v", maxAllocs, key), maxAllocs, func() {
			_, errSink = ctx.Call(bctx, impl, key)
		})
	}
}

// ReaderLatencyWithin enforces a per-call latency budget and fails
// the benchmark if the measured per-call latency exceeds maxLatency.
// Deterministic gate suitable for CI.
func ReaderLatencyWithin[T any, K comparable, V any](key K, maxLatency time.Duration) Reader[T, K, V] {
	return func(ctx ReaderContext[T, K, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		LatencyWithin(ctx.B, fmt.Sprintf("latency-within-%v/%v", maxLatency, key), maxLatency, func() {
			_, errSink = ctx.Call(bctx, impl, key)
		})
	}
}

// ReaderConcurrentThroughput measures read throughput under contention.
// Uses b.RunParallel for correct iteration scaling and goroutine lifecycle.
// Reports ns/op and allocs/op. Measurement only — does not fail the benchmark.
func ReaderConcurrentThroughput[T any, K comparable, V any](key K, parallelism int) Reader[T, K, V] {
	return func(ctx ReaderContext[T, K, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		ConcurrentThroughput(ctx.B, fmt.Sprintf("concurrent-%d/%v", parallelism, key), parallelism, func() {
			_, errSink = ctx.Call(bctx, impl, key)
		})
	}
}
