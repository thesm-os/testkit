// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bindings"
)

// ReaderNoErrorContext provides a typed factory and call function to
// ReaderNoError-shape bench primitives. A ReaderNoError-shaped method
// has the signature `func(ctx?, K) V` — infallible lookups against
// in-memory state (caches, gauges, stable mappings).
type ReaderNoErrorContext[T any, K comparable, V any] struct {
	B *testing.B
	bindings.ReaderNoErrorBindings[T, K, V]
}

// ReaderNoError is a typed bench primitive for ReaderNoError-shaped methods.
type ReaderNoError[T any, K comparable, V any] func(ReaderNoErrorContext[T, K, V])

// ReaderNoErrorHotPath measures the single-goroutine infallible-read
// latency and allocation rate for the given key.
func ReaderNoErrorHotPath[T any, K comparable, V any](key K) ReaderNoError[T, K, V] {
	return func(ctx ReaderNoErrorContext[T, K, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		HotPath(ctx.B, "hot-path/"+SubtestKey(key), func() {
			_ = ctx.Call(bctx, impl, key)
		})
	}
}

// ReaderNoErrorAllocsWithin measures allocations per infallible read
// and fails the benchmark if allocs exceed maxAllocs.
func ReaderNoErrorAllocsWithin[T any, K comparable, V any](key K, maxAllocs int) ReaderNoError[T, K, V] {
	return func(ctx ReaderNoErrorContext[T, K, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		AllocsWithin(ctx.B, fmt.Sprintf("allocs-within-%d/%s", maxAllocs, SubtestKey(key)), maxAllocs, func() {
			_ = ctx.Call(bctx, impl, key)
		})
	}
}

// ReaderNoErrorLatencyWithin enforces a per-call latency budget and
// fails the benchmark if the measured per-call latency exceeds
// maxLatency.
func ReaderNoErrorLatencyWithin[T any, K comparable, V any](key K, maxLatency time.Duration) ReaderNoError[T, K, V] {
	return func(ctx ReaderNoErrorContext[T, K, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		LatencyWithin(ctx.B, fmt.Sprintf("latency-within-%v/%s", maxLatency, SubtestKey(key)), maxLatency, func() {
			_ = ctx.Call(bctx, impl, key)
		})
	}
}

// ReaderNoErrorConcurrentThroughput measures infallible-read
// throughput under contention. Uses b.RunParallel for correct
// iteration scaling.
func ReaderNoErrorConcurrentThroughput[T any, K comparable, V any](key K, parallelism int) ReaderNoError[T, K, V] {
	return func(ctx ReaderNoErrorContext[T, K, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		ConcurrentThroughput(ctx.B, fmt.Sprintf("concurrent-%d/%s", parallelism, SubtestKey(key)), parallelism, func() {
			_ = ctx.Call(bctx, impl, key)
		})
	}
}
