// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"

	"go.thesmos.sh/testkit/bindings"
)

// DeleterContext provides a typed factory and call function to
// Deleter-shape bench primitives.
type DeleterContext[T any, K comparable] struct {
	B *testing.B
	bindings.DeleterBindings[T, K]
}

// Deleter is a typed bench primitive for Deleter-shaped methods.
type Deleter[T any, K comparable] func(DeleterContext[T, K])

// DeleterHotPath measures the single-goroutine delete latency and
// allocation rate for the given key. Reports ns/op and allocs/op.
func DeleterHotPath[T any, K comparable](key K) Deleter[T, K] {
	return func(ctx DeleterContext[T, K]) {
		ctx.B.Run(fmt.Sprintf("hot-path/%v", key), func(b *testing.B) {
			impl := ctx.Factory()
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				_ = ctx.Call(b.Context(), impl, key)
			}
		})
	}
}

// DeleterAllocsWithin measures allocations per delete and fails the
// benchmark if allocs exceed maxAllocs.
func DeleterAllocsWithin[T any, K comparable](key K, maxAllocs int) Deleter[T, K] {
	return func(ctx DeleterContext[T, K]) {
		ctx.B.Run(fmt.Sprintf("allocs-within-%d/%v", maxAllocs, key), func(b *testing.B) {
			allocs := testing.AllocsPerRun(100, func() {
				impl := ctx.Factory()
				_ = ctx.Call(b.Context(), impl, key)
			})
			if int(allocs) > maxAllocs {
				b.Fatalf("deleter allocs %d exceeds budget %d", int(allocs), maxAllocs)
			}
		})
	}
}
