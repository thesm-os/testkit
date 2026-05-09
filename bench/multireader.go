// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bindings"
)

// MultiReaderContext provides a typed factory and call function to
// MultiReader-shape bench primitives. A MultiReader-shaped method has
// the signature `func(ctx?, K) (V1, V2, error)` — get the entity +
// metadata idioms.
type MultiReaderContext[T any, K comparable, V1, V2 any] struct {
	B *testing.B
	bindings.MultiReaderBindings[T, K, V1, V2]
}

// MultiReader is a typed bench primitive for MultiReader-shaped methods.
type MultiReader[T any, K comparable, V1, V2 any] func(MultiReaderContext[T, K, V1, V2])

// MultiReaderHotPath measures the single-goroutine multi-read latency
// and allocation rate for the given key. Reports ns/op and allocs/op.
func MultiReaderHotPath[T any, K comparable, V1, V2 any](key K) MultiReader[T, K, V1, V2] {
	return func(ctx MultiReaderContext[T, K, V1, V2]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		HotPath(ctx.B, fmt.Sprintf("hot-path/%v", key), func() {
			_, _, errSink = ctx.Call(bctx, impl, key)
		})
	}
}

// MultiReaderAllocsWithin measures allocations per multi-read and
// fails the benchmark if allocs exceed maxAllocs.
func MultiReaderAllocsWithin[T any, K comparable, V1, V2 any](key K, maxAllocs int) MultiReader[T, K, V1, V2] {
	return func(ctx MultiReaderContext[T, K, V1, V2]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		AllocsWithin(ctx.B, fmt.Sprintf("allocs-within-%d/%v", maxAllocs, key), maxAllocs, func() {
			_, _, errSink = ctx.Call(bctx, impl, key)
		})
	}
}

// MultiReaderLatencyWithin enforces a per-call latency budget and
// fails the benchmark if the measured per-call latency exceeds
// maxLatency.
func MultiReaderLatencyWithin[T any, K comparable, V1, V2 any](
	key K, maxLatency time.Duration,
) MultiReader[T, K, V1, V2] {
	return func(ctx MultiReaderContext[T, K, V1, V2]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		name := fmt.Sprintf("latency-within-%v/%v", maxLatency, key)
		LatencyWithin(ctx.B, name, maxLatency, func() {
			_, _, errSink = ctx.Call(bctx, impl, key)
		})
	}
}

// MultiReaderConcurrentThroughput measures multi-read throughput under
// contention. Uses b.RunParallel for correct iteration scaling.
func MultiReaderConcurrentThroughput[T any, K comparable, V1, V2 any](
	key K, parallelism int,
) MultiReader[T, K, V1, V2] {
	return func(ctx MultiReaderContext[T, K, V1, V2]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		name := fmt.Sprintf("concurrent-%d/%v", parallelism, key)
		ConcurrentThroughput(ctx.B, name, parallelism, func() {
			_, _, errSink = ctx.Call(bctx, impl, key)
		})
	}
}
