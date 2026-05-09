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
//
// BytesPerOp records bytes processed per batch (e.g. byte size of
// the returned []V slice). When non-zero, primitives call
// b.SetBytes so `go test -bench` reports MB/s alongside ns/op —
// useful for comparing impls on payload-size-bound workloads.
type BatchReaderContext[T any, K comparable, V any] struct {
	B *testing.B
	bindings.BatchReaderBindings[T, K, V]
	BytesPerOp int64
}

// BatchReader is a typed bench primitive for BatchReader-shaped methods.
type BatchReader[T any, K comparable, V any] func(BatchReaderContext[T, K, V])

// BatchReaderHotPath measures the single-goroutine batch-read latency
// and allocation rate for the given key set. Reports ns/op and allocs/op.
func BatchReaderHotPath[T any, K comparable, V any](keys []K) BatchReader[T, K, V] {
	return func(ctx BatchReaderContext[T, K, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		HotPathWithBytes(ctx.B, fmt.Sprintf("hot-path/n=%d", len(keys)), ctx.BytesPerOp, func() {
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
		name := fmt.Sprintf("allocs-within-%d/n=%d", maxAllocs, len(keys))
		AllocsWithinWithBytes(ctx.B, name, ctx.BytesPerOp, maxAllocs, func() {
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
		name := fmt.Sprintf("latency-within-%v/n=%d", maxLatency, len(keys))
		LatencyWithinWithBytes(ctx.B, name, ctx.BytesPerOp, maxLatency, func() {
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
		name := fmt.Sprintf("concurrent-%d/n=%d", parallelism, len(keys))
		ConcurrentThroughputWithBytes(ctx.B, name, ctx.BytesPerOp, parallelism, func() {
			_, errSink = ctx.Call(bctx, impl, keys)
		})
	}
}
