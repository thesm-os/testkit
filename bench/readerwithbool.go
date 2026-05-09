// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bindings"
)

// ReaderWithBoolContext provides a typed factory and call function to
// ReaderWithBool-shape bench primitives.
type ReaderWithBoolContext[T any, K comparable, V any] struct {
	B *testing.B
	bindings.ReaderWithBoolBindings[T, K, V]
}

// ReaderWithBool is a typed bench primitive for ReaderWithBool-shaped methods.
type ReaderWithBool[T any, K comparable, V any] func(ReaderWithBoolContext[T, K, V])

// ReaderWithBoolHotPath measures read throughput against a known key.
// Reports ns/op and allocs/op.
func ReaderWithBoolHotPath[T any, K comparable, V any](key K) ReaderWithBool[T, K, V] {
	return func(ctx ReaderWithBoolContext[T, K, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		HotPath(ctx.B, "hot-path/"+SubtestKey(key), func() {
			_, _ = ctx.Call(bctx, impl, key)
		})
	}
}

// ReaderWithBoolAllocsWithin measures allocations per read and fails
// the benchmark if allocs exceed maxAllocs.
func ReaderWithBoolAllocsWithin[T any, K comparable, V any](key K, maxAllocs int) ReaderWithBool[T, K, V] {
	return func(ctx ReaderWithBoolContext[T, K, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		AllocsWithin(ctx.B, fmt.Sprintf("allocs-within-%d/%s", maxAllocs, SubtestKey(key)), maxAllocs, func() {
			_, _ = ctx.Call(bctx, impl, key)
		})
	}
}

// ReaderWithBoolLatencyWithin enforces a per-call latency budget and
// fails the benchmark if the measured per-call latency exceeds
// maxLatency.
func ReaderWithBoolLatencyWithin[T any, K comparable, V any](key K, maxLatency time.Duration) ReaderWithBool[T, K, V] {
	return func(ctx ReaderWithBoolContext[T, K, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		LatencyWithin(ctx.B, fmt.Sprintf("latency-within-%v/%s", maxLatency, SubtestKey(key)), maxLatency, func() {
			_, _ = ctx.Call(bctx, impl, key)
		})
	}
}

// ReaderWithBoolConcurrentThroughput measures read throughput under
// contention. Uses b.RunParallel for correct iteration scaling.
func ReaderWithBoolConcurrentThroughput[T any, K comparable, V any](key K, parallelism int) ReaderWithBool[T, K, V] {
	return func(ctx ReaderWithBoolContext[T, K, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		ConcurrentThroughput(ctx.B, fmt.Sprintf("concurrent-%d/%s", parallelism, SubtestKey(key)), parallelism, func() {
			_, _ = ctx.Call(bctx, impl, key)
		})
	}
}
