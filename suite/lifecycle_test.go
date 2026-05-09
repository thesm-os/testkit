// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"go.thesmos.sh/testkit/bindings"
	"go.thesmos.sh/testkit/suite"
)

type lifecycle struct {
	mu     sync.Mutex
	opened bool
}

func newLifecycle() *lifecycle { return &lifecycle{} }

func (l *lifecycle) Open(_ context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.opened = true
	return nil
}

func (l *lifecycle) Close(ctx context.Context) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.opened {
		return errors.New("not opened")
	}
	l.opened = false
	return nil
}

var errLifecycleInvalid = errors.New("lifecycle: invalid")

func lifecycleCtx(t *testing.T) suite.LifecycleContext[*lifecycle] {
	t.Helper()
	return suite.LifecycleContext[*lifecycle]{
		T: t,
		LifecycleBindings: bindings.LifecycleBindings[*lifecycle]{
			Factory: newLifecycle,
			Call: func(ctx context.Context, l *lifecycle) error {
				return l.Open(ctx)
			},
		},
	}
}

func TestLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("Succeeds invokes the lifecycle entry point", func(t *testing.T) {
		t.Parallel()
		suite.AssertLifecycleSucceeds[*lifecycle]()(lifecycleCtx(t))
	})

	t.Run("Idempotent yields the same outcome across two calls", func(t *testing.T) {
		t.Parallel()
		suite.AssertLifecycleIdempotent[*lifecycle]()(lifecycleCtx(t))
	})

	t.Run("RespectsContext surfaces ctx.Canceled on cancelled call", func(t *testing.T) {
		t.Parallel()
		// Close-bound context: Close honors ctx.Err so the cancellation
		// surfaces as expected.
		ctx := suite.LifecycleContext[*lifecycle]{
			T: t,
			LifecycleBindings: bindings.LifecycleBindings[*lifecycle]{
				Factory: func() *lifecycle { l := newLifecycle(); l.opened = true; return l },
				Call: func(ctx context.Context, l *lifecycle) error {
					return l.Close(ctx)
				},
			},
		}
		suite.AssertLifecycleRespectsContext[*lifecycle]()(ctx)
	})

	t.Run("RejectInvalid surfaces the sentinel against an invalid factory", func(t *testing.T) {
		t.Parallel()
		ctx := suite.LifecycleContext[*lifecycle]{
			T: t,
			LifecycleBindings: bindings.LifecycleBindings[*lifecycle]{
				Factory: func() *lifecycle { return &lifecycle{} },
				Call: func(_ context.Context, l *lifecycle) error {
					if !l.opened {
						return errLifecycleInvalid
					}
					return nil
				},
			},
		}
		invalid := func() *lifecycle { return &lifecycle{} }
		suite.AssertLifecycleRejectInvalid[*lifecycle](invalid, errLifecycleInvalid)(ctx)
	})

	t.Run("ConcurrentSafe runs without races", func(t *testing.T) {
		t.Parallel()
		suite.AssertLifecycleConcurrentSafe[*lifecycle](4, 50)(lifecycleCtx(t))
	})
}
