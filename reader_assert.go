// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// ReaderContext provides typed domain inputs and a typed call function to
// Reader-shape primitives. Populated by generator-emitted options.
//
// A Reader-shaped method has the signature func(ctx, K) (V, error).
type ReaderContext[T any, K comparable, V any] struct {
	T       *testing.T
	Factory func() T
	Call    func(T, context.Context, K) (V, error)

	// Known keys — populated by Known(...) option.
	Known []K

	// Unknown keys — populated by Unknown(...) option.
	Unknown []K

	// Want maps — populated by Expect(K, V) option.
	Want map[K]V
}

// ReaderAssertion is a typed conformance primitive for Reader-shaped methods.
type ReaderAssertion[T any, K comparable, V any] func(ReaderContext[T, K, V])

// AssertReturnsForKey reads each (key, want) pair from ctx.Want and
// asserts the reader returns the expected value. One subtest per key
// for failure isolation.
func AssertReturnsForKey[T any, K comparable, V any]() ReaderAssertion[T, K, V] {
	return func(ctx ReaderContext[T, K, V]) {
		ctx.T.Run("returns for key", func(t *testing.T) {
			if len(ctx.Want) == 0 {
				t.Skip("AssertReturnsForKey: no Expect(...) options wired")
			}
			for k, want := range ctx.Want {
				t.Run(fmt.Sprintf("%v", k), func(t *testing.T) {
					t.Parallel()
					impl := ctx.Factory()
					got, err := ctx.Call(impl, t.Context(), k)
					NoError(t, err, "reader must not error for known key")
					Equal(t, got, want, "reader must return expected value")
				})
			}
		})
	}
}

// AssertReturnsSentinel reads each key from ctx.Unknown and asserts the
// reader returns the given sentinel error. One subtest per key.
func AssertReturnsSentinel[T any, K comparable, V any](sentinel error) ReaderAssertion[T, K, V] {
	return func(ctx ReaderContext[T, K, V]) {
		ctx.T.Run("returns sentinel for unknown", func(t *testing.T) {
			if len(ctx.Unknown) == 0 {
				t.Skip("AssertReturnsSentinel: no Unknown(...) options wired")
			}
			for _, k := range ctx.Unknown {
				t.Run(fmt.Sprintf("%v", k), func(t *testing.T) {
					t.Parallel()
					impl := ctx.Factory()
					_, err := ctx.Call(impl, t.Context(), k)
					ErrorIs(t, err, sentinel, "reader must return sentinel for unknown key")
				})
			}
		})
	}
}

// AssertConsistentReads calls the reader N times with the same key and
// asserts all results are equal. Requires at least one Known key and n >= 2.
func AssertConsistentReads[T any, K comparable, V any](n int) ReaderAssertion[T, K, V] {
	return func(ctx ReaderContext[T, K, V]) {
		ctx.T.Run("consistent reads", func(t *testing.T) {
			if len(ctx.Known) == 0 {
				t.Skip("AssertConsistentReads: no Known(...) options wired")
			}
			if n < 2 {
				t.Fatalf("AssertConsistentReads: n must be >= 2, got %d", n)
			}
			t.Parallel()
			impl := ctx.Factory()
			k := ctx.Known[0]
			first, err := ctx.Call(impl, t.Context(), k)
			NoError(t, err, "first read must not error")
			for i := 1; i < n; i++ {
				got, err := ctx.Call(impl, t.Context(), k)
				NoError(t, err, "read must not error")
				Equal(t, got, first, "read must be consistent")
			}
		})
	}
}

// AssertReadsAreNonMutating calls observe before and after a read,
// asserts the observable state did not change.
func AssertReadsAreNonMutating[T any, K comparable, V any, S comparable](
	observe func(T) S,
) ReaderAssertion[T, K, V] {
	return func(ctx ReaderContext[T, K, V]) {
		ctx.T.Run("reads are non-mutating", func(t *testing.T) {
			if len(ctx.Known) == 0 {
				t.Skip("AssertReadsAreNonMutating: no Known(...) options wired")
			}
			t.Parallel()
			impl := ctx.Factory()
			before := observe(impl)
			_, _ = ctx.Call(impl, t.Context(), ctx.Known[0])
			after := observe(impl)
			Equal(t, before, after, "read must not mutate observable state")
		})
	}
}

// AssertReaderConcurrentSafe runs the reader from N goroutines concurrently.
// Panics propagate (Go default); race detector finds data races when -race
// is enabled.
func AssertReaderConcurrentSafe[T any, K comparable, V any](
	workers, iters int,
) ReaderAssertion[T, K, V] {
	return func(ctx ReaderContext[T, K, V]) {
		ctx.T.Run("concurrent safe", func(t *testing.T) {
			if len(ctx.Known) == 0 {
				t.Skip("AssertReaderConcurrentSafe: no Known(...) options wired")
			}
			t.Parallel()
			impl := ctx.Factory()
			k := ctx.Known[0]
			var wg sync.WaitGroup
			for range workers {
				wg.Go(func() {
					for range iters {
						_, _ = ctx.Call(impl, t.Context(), k)
					}
				})
			}
			wg.Wait()
		})
	}
}
