// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/bindings"
	"go.thesmos.sh/testkit/suite"
)

type resetter struct {
	mu      sync.Mutex
	calls   atomic.Int32
	corrupt bool
}

func (r *resetter) Reset(_ context.Context) {
	r.calls.Add(1)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.corrupt = false
}

func voidLifecycleCtx(t *testing.T) suite.VoidLifecycleContext[*resetter] {
	t.Helper()
	return suite.VoidLifecycleContext[*resetter]{
		T: t,
		VoidLifecycleBindings: bindings.VoidLifecycleBindings[*resetter]{
			Factory: func() *resetter { return &resetter{} },
			Call: func(ctx context.Context, r *resetter) {
				r.Reset(ctx)
			},
		},
	}
}

func TestVoidLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("Succeeds without panic", func(t *testing.T) {
		t.Parallel()
		suite.AssertVoidLifecycleSucceeds[*resetter]()(voidLifecycleCtx(t))
	})

	t.Run("Idempotent across two calls", func(t *testing.T) {
		t.Parallel()
		suite.AssertVoidLifecycleIdempotent[*resetter]()(voidLifecycleCtx(t))
	})

	t.Run("RespectsContext under cancelled ctx", func(t *testing.T) {
		t.Parallel()
		suite.AssertVoidLifecycleRespectsContext[*resetter]()(voidLifecycleCtx(t))
	})

	t.Run("RejectInvalid runs the consumer's check after the call", func(t *testing.T) {
		t.Parallel()
		invalidFactory := func() *resetter { return &resetter{corrupt: true} }
		check := func(t *testing.T, r *resetter) {
			t.Helper()
			testkit.False(t, r.corrupt, "Reset must clear the corrupt flag")
		}
		suite.AssertVoidLifecycleRejectInvalid[*resetter](invalidFactory, check)(
			voidLifecycleCtx(t),
		)
	})

	t.Run("RejectInvalidWith does not panic on invalid impl", func(t *testing.T) {
		t.Parallel()
		invalid := func() *resetter { return &resetter{corrupt: true} }
		suite.AssertVoidLifecycleRejectInvalidWith[*resetter](invalid)(voidLifecycleCtx(t))
	})

	t.Run("ConcurrentSafe runs without races", func(t *testing.T) {
		t.Parallel()
		suite.AssertVoidLifecycleConcurrentSafe[*resetter](4, 50)(voidLifecycleCtx(t))
	})
}

func TestAssertVoidLifecycleBaseline(t *testing.T) {
	t.Parallel()
	suite.AssertVoidLifecycleBaseline[*resetter]()(voidLifecycleCtx(t))
}
