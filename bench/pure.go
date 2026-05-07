// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"

	"go.thesmos.sh/testkit/bindings"
)

// PureContext provides a typed factory and call function to
// Pure-shape bench primitives.
type PureContext[T, R any] struct {
	B *testing.B
	bindings.PureBindings[T, R]
}

// Pure is a typed bench primitive for Pure-shaped methods.
type Pure[T, R any] func(PureContext[T, R])

// PureConcurrentThroughput measures pure-method throughput under
// contention. Uses b.RunParallel for correct iteration scaling.
// Reports ns/op and allocs/op. Measurement only — does not fail.
func PureConcurrentThroughput[T, R any](parallelism int) Pure[T, R] {
	return func(ctx PureContext[T, R]) {
		ctx.B.Run(fmt.Sprintf("concurrent-%d", parallelism), func(b *testing.B) {
			impl := ctx.Factory()
			b.SetParallelism(parallelism)
			b.ResetTimer()
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = ctx.Call(impl)
				}
			})
		})
	}
}

// PureAllocsWithin measures allocations per pure call and fails
// the benchmark if allocs exceed maxAllocs.
func PureAllocsWithin[T, R any](maxAllocs int) Pure[T, R] {
	return func(ctx PureContext[T, R]) {
		ctx.B.Run("allocs-within", func(b *testing.B) {
			impl := ctx.Factory()
			allocs := testing.AllocsPerRun(100, func() {
				_ = ctx.Call(impl)
			})
			if int(allocs) > maxAllocs {
				b.Fatalf("pure allocs %d exceeds budget %d", int(allocs), maxAllocs)
			}
		})
	}
}
