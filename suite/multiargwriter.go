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

// MultiArgWriterContext provides a typed factory and call function to
// MultiArgWriter-shape primitives. A MultiArgWriter-shaped method has
// the signature `func(ctx, P1, P2, P3) error` — 3 non-ctx parameters.
// Methods with 4+ non-ctx params use a consumer-supplied plug-in.
type MultiArgWriterContext[T any, P1, P2, P3 any] struct {
	T *testing.T
	bindings.MultiArgWriterBindings[T, P1, P2, P3]
}

// MultiArgWriterAssertion is a typed conformance primitive for
// MultiArgWriter-shaped methods.
type MultiArgWriterAssertion[T any, P1, P2, P3 any] func(MultiArgWriterContext[T, P1, P2, P3])

// AssertMultiArgWriteSucceeds writes the given args and asserts no error.
func AssertMultiArgWriteSucceeds[T, P1, P2, P3 any](
	p1 P1,
	p2 P2,
	p3 P3,
) MultiArgWriterAssertion[T, P1, P2, P3] {
	return func(ctx MultiArgWriterContext[T, P1, P2, P3]) {
		ctx.T.Run("write succeeds", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(t.Context(), impl, p1, p2, p3)
			testkit.NoError(t, err, "multi-arg write must succeed")
		})
	}
}

// AssertMultiArgWriteRejectInvalid writes invalid args and asserts the
// configured sentinel is returned.
func AssertMultiArgWriteRejectInvalid[T, P1, P2, P3 any](
	p1 P1,
	p2 P2,
	p3 P3,
	sentinel error,
) MultiArgWriterAssertion[T, P1, P2, P3] {
	return func(ctx MultiArgWriterContext[T, P1, P2, P3]) {
		ctx.T.Run("rejects invalid", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(t.Context(), impl, p1, p2, p3)
			testkit.ErrorIs(t, err, sentinel,
				"multi-arg writer must surface sentinel for invalid args")
		})
	}
}

// AssertMultiArgWriterIdempotent writes the same args twice and asserts
// both calls succeed.
func AssertMultiArgWriterIdempotent[T, P1, P2, P3 any](
	p1 P1,
	p2 P2,
	p3 P3,
) MultiArgWriterAssertion[T, P1, P2, P3] {
	return func(ctx MultiArgWriterContext[T, P1, P2, P3]) {
		ctx.T.Run("idempotent", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(t.Context(), impl, p1, p2, p3)
			testkit.NoError(t, err, "first write must succeed")
			err = ctx.Call(t.Context(), impl, p1, p2, p3)
			testkit.NoError(t, err, "second write of same args must succeed (idempotent)")
		})
	}
}

// AssertMultiArgWriterRespectsContext invokes the writer with a cancelled
// context and asserts context.Canceled is returned.
func AssertMultiArgWriterRespectsContext[T, P1, P2, P3 any](
	p1 P1,
	p2 P2,
	p3 P3,
) MultiArgWriterAssertion[T, P1, P2, P3] {
	return func(ctx MultiArgWriterContext[T, P1, P2, P3]) {
		ctx.T.Run("respects context", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			cctx, cancel := context.WithCancel(t.Context())
			cancel()
			err := ctx.Call(cctx, impl, p1, p2, p3)
			testkit.ErrorIs(t, err, context.Canceled,
				"multi-arg writer must surface ctx.Canceled when called with a cancelled context")
		})
	}
}

// AssertMultiArgWriterConcurrentSafe runs the writer from N goroutines
// concurrently.
func AssertMultiArgWriterConcurrentSafe[T, P1, P2, P3 any](
	p1 P1,
	p2 P2,
	p3 P3,
	workers, iters int,
) MultiArgWriterAssertion[T, P1, P2, P3] {
	return func(ctx MultiArgWriterContext[T, P1, P2, P3]) {
		ctx.T.Run("concurrent safe", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			var wg sync.WaitGroup
			for range workers {
				wg.Go(func() {
					for range iters {
						_ = ctx.Call(t.Context(), impl, p1, p2, p3)
					}
				})
			}
			wg.Wait()
		})
	}
}
