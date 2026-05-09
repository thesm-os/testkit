// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"context"
	"sync"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/bindings"
)

// MultiAggregatorContext provides a typed factory and call function to
// MultiAggregator-shape primitives. A MultiAggregator-shaped method has
// the signature `func(ctx?) (V1, V2, error)` — 2-tuple aggregations.
type MultiAggregatorContext[T any, V1, V2 any] struct {
	T *testing.T
	bindings.MultiAggregatorBindings[T, V1, V2]
}

// MultiAggregatorAssertion is a typed conformance primitive for
// MultiAggregator-shaped methods.
type MultiAggregatorAssertion[T any, V1, V2 any] func(MultiAggregatorContext[T, V1, V2])

// AssertMultiAggregatorReturns calls the aggregator and asserts both
// return values match.
func AssertMultiAggregatorReturns[T any, V1, V2 comparable](
	want1 V1,
	want2 V2,
) MultiAggregatorAssertion[T, V1, V2] {
	return func(ctx MultiAggregatorContext[T, V1, V2]) {
		ctx.T.Run("returns expected", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			got1, got2, err := ctx.Call(t.Context(), impl)
			testkit.NoError(t, err, "multi-aggregator must not error")
			testkit.Equal(t, got1, want1, "first return value must match")
			testkit.Equal(t, got2, want2, "second return value must match")
		})
	}
}

// AssertMultiAggregatorReturnsSentinel calls the aggregator on a known-
// invalid factory and asserts the configured sentinel is returned.
func AssertMultiAggregatorReturnsSentinel[T, V1, V2 any](
	invalidFactory func() T,
	sentinel error,
) MultiAggregatorAssertion[T, V1, V2] {
	return func(ctx MultiAggregatorContext[T, V1, V2]) {
		ctx.T.Run("returns sentinel", func(t *testing.T) {
			t.Parallel()
			impl := invalidFactory()
			_, _, err := ctx.Call(t.Context(), impl)
			testkit.ErrorIs(t, err, sentinel,
				"multi-aggregator must surface sentinel against an invalid-state impl")
		})
	}
}

// AssertMultiAggregatorConsistent calls the aggregator N times and
// asserts both return values match across calls.
func AssertMultiAggregatorConsistent[T any, V1, V2 comparable](
	n int,
) MultiAggregatorAssertion[T, V1, V2] {
	return func(ctx MultiAggregatorContext[T, V1, V2]) {
		ctx.T.Run("consistent", func(t *testing.T) {
			if n < 2 {
				t.Fatalf("AssertMultiAggregatorConsistent: n must be >= 2, got %d", n)
			}
			t.Parallel()
			impl := ctx.Factory()
			first1, first2, err := ctx.Call(t.Context(), impl)
			testkit.NoError(t, err, "first call must not error")
			for i := 1; i < n; i++ {
				got1, got2, err := ctx.Call(t.Context(), impl)
				testkit.NoError(t, err, "call must not error")
				testkit.Equal(t, got1, first1, "first value must be consistent")
				testkit.Equal(t, got2, first2, "second value must be consistent")
			}
		})
	}
}

// AssertMultiAggregatorRespectsContext invokes the aggregator with a
// cancelled context and asserts context.Canceled is returned.
func AssertMultiAggregatorRespectsContext[T, V1, V2 any]() MultiAggregatorAssertion[T, V1, V2] {
	return func(ctx MultiAggregatorContext[T, V1, V2]) {
		ctx.T.Run("respects context", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			cctx, cancel := context.WithCancel(t.Context())
			cancel()
			_, _, err := ctx.Call(cctx, impl)
			testkit.ErrorIs(t, err, context.Canceled,
				"multi-aggregator must surface ctx.Canceled when called with a cancelled context")
		})
	}
}

// AssertMultiAggregatorConcurrentSafe runs the aggregator from N
// goroutines concurrently.
func AssertMultiAggregatorConcurrentSafe[T, V1, V2 any](
	workers, iters int,
) MultiAggregatorAssertion[T, V1, V2] {
	return func(ctx MultiAggregatorContext[T, V1, V2]) {
		ctx.T.Run("concurrent safe", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			var wg sync.WaitGroup
			for range workers {
				wg.Go(func() {
					for range iters {
						_, _, _ = ctx.Call(t.Context(), impl)
					}
				})
			}
			wg.Wait()
		})
	}
}

// AssertMultiAggregatorSmoke calls the aggregator once on a fresh impl.
// The subtest fails fast on panic, surfacing a broken Factory or a
// method that panics on bare invocation as one localized failure before
// any contract assertion runs.
func AssertMultiAggregatorSmoke[T, V1, V2 any]() MultiAggregatorAssertion[T, V1, V2] {
	return func(ctx MultiAggregatorContext[T, V1, V2]) {
		ctx.T.Run("smoke", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			_, _, _ = ctx.Call(t.Context(), impl)
		})
	}
}

// AssertMultiAggregatorBaseline runs the MultiAggregator-shape baseline:
// smoke, Returns(wantV1, wantV2), RespectsContext, Consistent over 3
// calls, and ConcurrentSafe (4×10). Optional extras (e.g.
// ReturnsSentinel under an InvalidFactory) run between consistency and
// concurrency.
func AssertMultiAggregatorBaseline[T any, V1, V2 comparable](
	wantV1 V1,
	wantV2 V2,
	extra ...MultiAggregatorAssertion[T, V1, V2],
) MultiAggregatorAssertion[T, V1, V2] {
	return func(ctx MultiAggregatorContext[T, V1, V2]) {
		AssertMultiAggregatorSmoke[T, V1, V2]()(ctx)
		AssertMultiAggregatorReturns[T, V1, V2](wantV1, wantV2)(ctx)
		AssertMultiAggregatorRespectsContext[T, V1, V2]()(ctx)
		AssertMultiAggregatorConsistent[T, V1, V2](3)(ctx)
		for _, e := range extra {
			e(ctx)
		}
		AssertMultiAggregatorConcurrentSafe[T, V1, V2](4, 10)(ctx)
	}
}
