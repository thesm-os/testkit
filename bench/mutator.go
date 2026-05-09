// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bindings"
)

// MutatorContext provides a typed factory and call function to
// Mutator-shape bench primitives.
type MutatorContext[T, V any] struct {
	B *testing.B
	bindings.MutatorBindings[T, V]
}

// Mutator is a typed bench primitive for Mutator-shaped methods.
type Mutator[T, V any] func(MutatorContext[T, V])

// MutatorHotPath measures mutator throughput with a sample value.
// Repeated calls accumulate state in the impl; benchmarks should
// pick a value whose mutation is idempotent or use a factory that
// reinitializes between primitive invocations.
func MutatorHotPath[T, V any](sample V) Mutator[T, V] {
	return func(ctx MutatorContext[T, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		HotPath(ctx.B, fmt.Sprintf("hot-path/%v", sample), func() {
			ctx.Call(bctx, impl, sample)
		})
	}
}

// MutatorAllocsWithin measures allocations per mutator call and fails
// the benchmark if allocs exceed maxAllocs.
func MutatorAllocsWithin[T, V any](sample V, maxAllocs int) Mutator[T, V] {
	return func(ctx MutatorContext[T, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		AllocsWithin(ctx.B, fmt.Sprintf("allocs-within-%d/%v", maxAllocs, sample), maxAllocs, func() {
			ctx.Call(bctx, impl, sample)
		})
	}
}

// MutatorLatencyWithin enforces a per-call latency budget and fails
// the benchmark if the measured per-call latency exceeds maxLatency.
func MutatorLatencyWithin[T, V any](sample V, maxLatency time.Duration) Mutator[T, V] {
	return func(ctx MutatorContext[T, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		LatencyWithin(ctx.B, fmt.Sprintf("latency-within-%v/%v", maxLatency, sample), maxLatency, func() {
			ctx.Call(bctx, impl, sample)
		})
	}
}

// MutatorConcurrentThroughput measures mutator throughput under
// contention. Uses b.RunParallel for correct iteration scaling.
func MutatorConcurrentThroughput[T, V any](sample V, parallelism int) Mutator[T, V] {
	return func(ctx MutatorContext[T, V]) {
		impl := ctx.Factory()
		bctx := ctx.B.Context()
		ConcurrentThroughput(ctx.B, fmt.Sprintf("concurrent-%d/%v", parallelism, sample), parallelism, func() {
			ctx.Call(bctx, impl, sample)
		})
	}
}
