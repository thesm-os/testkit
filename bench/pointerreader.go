// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bindings"
)

// PointerReaderContext provides a typed factory and call function to
// PointerReader-shape bench primitives. A PointerReader-shaped method
// has the signature `func(ctx?, K) *V` — the nil-on-miss idiom.
type PointerReaderContext[T any, K comparable, V any] struct {
	B *testing.B
	bindings.PointerReaderBindings[T, K, V]
}

// PointerReader is a typed bench primitive for PointerReader-shaped methods.
type PointerReader[T any, K comparable, V any] func(PointerReaderContext[T, K, V])

// PointerReaderHotPath measures the single-goroutine pointer-read
// latency and allocation rate for the given key.
func PointerReaderHotPath[T any, K comparable, V any](key K) PointerReader[T, K, V] {
	return func(ctx PointerReaderContext[T, K, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		HotPath(ctx.B, "hot-path/"+SubtestKey(key), func() {
			_ = ctx.Call(bctx, impl, key)
		})
	}
}

// PointerReaderAllocsWithin measures allocations per pointer-read and
// fails the benchmark if allocs exceed maxAllocs.
func PointerReaderAllocsWithin[T any, K comparable, V any](key K, maxAllocs int) PointerReader[T, K, V] {
	return func(ctx PointerReaderContext[T, K, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		AllocsWithin(ctx.B, fmt.Sprintf("allocs-within-%d/%s", maxAllocs, SubtestKey(key)), maxAllocs, func() {
			_ = ctx.Call(bctx, impl, key)
		})
	}
}

// PointerReaderLatencyWithin enforces a per-call latency budget and
// fails the benchmark if the measured per-call latency exceeds
// maxLatency.
func PointerReaderLatencyWithin[T any, K comparable, V any](key K, maxLatency time.Duration) PointerReader[T, K, V] {
	return func(ctx PointerReaderContext[T, K, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		LatencyWithin(ctx.B, fmt.Sprintf("latency-within-%v/%s", maxLatency, SubtestKey(key)), maxLatency, func() {
			_ = ctx.Call(bctx, impl, key)
		})
	}
}

// PointerReaderConcurrentThroughput measures pointer-read throughput
// under contention. Uses b.RunParallel for correct iteration scaling.
func PointerReaderConcurrentThroughput[T any, K comparable, V any](key K, parallelism int) PointerReader[T, K, V] {
	return func(ctx PointerReaderContext[T, K, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		ConcurrentThroughput(ctx.B, fmt.Sprintf("concurrent-%d/%s", parallelism, SubtestKey(key)), parallelism, func() {
			_ = ctx.Call(bctx, impl, key)
		})
	}
}
