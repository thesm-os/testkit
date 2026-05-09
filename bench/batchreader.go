// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bindings"
)

// BatchReaderContext provides a typed factory and call function to
// BatchReader-shape bench primitives. A BatchReader-shaped method has the
// signature `func(ctx?, ...K) ([]V, error)` — variadic-key fetch.
type BatchReaderContext[T any, K comparable, V any] struct {
	B *testing.B
	bindings.BatchReaderBindings[T, K, V]
}

// BatchReader is a typed bench primitive for BatchReader-shaped methods.
type BatchReader[T any, K comparable, V any] func(BatchReaderContext[T, K, V])

// BatchReaderHotPath measures the single-goroutine batch-read latency
// and allocation rate for the given key set. Reports ns/op and allocs/op.
func BatchReaderHotPath[T any, K comparable, V any](keys []K) BatchReader[T, K, V] {
	return func(ctx BatchReaderContext[T, K, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		HotPath(ctx.B, fmt.Sprintf("hot-path/n=%d", len(keys)), func() {
			_, errSink = ctx.Call(bctx, impl, keys)
		})
	}
}

// BatchReaderAllocsWithin measures allocations per batch read and
// fails the benchmark if allocs exceed maxAllocs.
func BatchReaderAllocsWithin[T any, K comparable, V any](keys []K, maxAllocs int) BatchReader[T, K, V] {
	return func(ctx BatchReaderContext[T, K, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		AllocsWithin(ctx.B, fmt.Sprintf("allocs-within-%d/n=%d", maxAllocs, len(keys)), maxAllocs, func() {
			_, errSink = ctx.Call(bctx, impl, keys)
		})
	}
}

// BatchReaderLatencyWithin enforces a per-call latency budget and
// fails the benchmark if the measured per-call latency exceeds
// maxLatency. Per-call here means a full batch fetch.
func BatchReaderLatencyWithin[T any, K comparable, V any](keys []K, maxLatency time.Duration) BatchReader[T, K, V] {
	return func(ctx BatchReaderContext[T, K, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		LatencyWithin(ctx.B, fmt.Sprintf("latency-within-%v/n=%d", maxLatency, len(keys)), maxLatency, func() {
			_, errSink = ctx.Call(bctx, impl, keys)
		})
	}
}

// BatchReaderConcurrentThroughput measures batch-read throughput under
// contention. Uses b.RunParallel for correct iteration scaling.
func BatchReaderConcurrentThroughput[T any, K comparable, V any](keys []K, parallelism int) BatchReader[T, K, V] {
	return func(ctx BatchReaderContext[T, K, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		ConcurrentThroughput(ctx.B, fmt.Sprintf("concurrent-%d/n=%d", parallelism, len(keys)), parallelism, func() {
			_, errSink = ctx.Call(bctx, impl, keys)
		})
	}
}
