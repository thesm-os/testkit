// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bindings"
)

// CompositeWriterContext provides a typed factory and call function
// to CompositeWriter-shape bench primitives. A CompositeWriter-shaped
// method has the signature `func(ctx?, K1, V) error` — namespaced
// stores, tagged caches, two-key indexes.
type CompositeWriterContext[T any, K1 comparable, V any] struct {
	B *testing.B
	bindings.CompositeWriterBindings[T, K1, V]
}

// CompositeWriter is a typed bench primitive for CompositeWriter-shaped methods.
type CompositeWriter[T any, K1 comparable, V any] func(CompositeWriterContext[T, K1, V])

// CompositeWriterHotPath measures the single-goroutine composite-write
// latency and allocation rate for the given (namespace, value) pair.
func CompositeWriterHotPath[T any, K1 comparable, V any](namespace K1, sample V) CompositeWriter[T, K1, V] {
	return func(ctx CompositeWriterContext[T, K1, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		HotPath(ctx.B, fmt.Sprintf("hot-path/%v/%v", namespace, sample), func() {
			errSink = ctx.Call(bctx, impl, namespace, sample)
		})
	}
}

// CompositeWriterAllocsWithin measures allocations per composite write
// and fails the benchmark if allocs exceed maxAllocs.
func CompositeWriterAllocsWithin[T any, K1 comparable, V any](
	namespace K1, sample V, maxAllocs int,
) CompositeWriter[T, K1, V] {
	return func(ctx CompositeWriterContext[T, K1, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		name := fmt.Sprintf("allocs-within-%d/%v/%v", maxAllocs, namespace, sample)
		AllocsWithin(ctx.B, name, maxAllocs, func() {
			errSink = ctx.Call(bctx, impl, namespace, sample)
		})
	}
}

// CompositeWriterLatencyWithin enforces a per-call latency budget and
// fails the benchmark if the measured per-call latency exceeds
// maxLatency.
func CompositeWriterLatencyWithin[T any, K1 comparable, V any](
	namespace K1, sample V, maxLatency time.Duration,
) CompositeWriter[T, K1, V] {
	return func(ctx CompositeWriterContext[T, K1, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		name := fmt.Sprintf("latency-within-%v/%v/%v", maxLatency, namespace, sample)
		LatencyWithin(ctx.B, name, maxLatency, func() {
			errSink = ctx.Call(bctx, impl, namespace, sample)
		})
	}
}

// CompositeWriterConcurrentThroughput measures composite-write
// throughput under contention. Uses b.RunParallel for correct
// iteration scaling.
func CompositeWriterConcurrentThroughput[T any, K1 comparable, V any](
	namespace K1, sample V, parallelism int,
) CompositeWriter[T, K1, V] {
	return func(ctx CompositeWriterContext[T, K1, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		name := fmt.Sprintf("concurrent-%d/%v/%v", parallelism, namespace, sample)
		ConcurrentThroughput(ctx.B, name, parallelism, func() {
			errSink = ctx.Call(bctx, impl, namespace, sample)
		})
	}
}
