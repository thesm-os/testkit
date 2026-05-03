// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"context"
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
)

type lifecycle struct{ opened bool }

func newLifecycle() *lifecycle { return &lifecycle{} }

func (l *lifecycle) Open(_ context.Context) error {
	l.opened = true
	return nil
}

func (l *lifecycle) Close(ctx context.Context) error {
	if ctx != nil {
		err := ctx.Err()
		if err != nil {
			return err
		}
	}
	if !l.opened {
		return errors.New("not opened")
	}
	l.opened = false
	return nil
}

func lifecycleCtx(t *testing.T) testkit.LifecycleContext[*lifecycle] {
	t.Helper()
	return testkit.LifecycleContext[*lifecycle]{
		T: t,
		LifecycleBindings: testkit.LifecycleBindings[*lifecycle]{
			Factory: newLifecycle,
			Call: func(ctx context.Context, l *lifecycle) error {
				return l.Open(ctx)
			},
		},
	}
}

func TestAssertLifecycleSucceeds(t *testing.T) {
	t.Parallel()
	ctx := lifecycleCtx(t)
	testkit.AssertLifecycleSucceeds[*lifecycle]()(ctx)
}

func TestAssertLifecycleIdempotent(t *testing.T) {
	t.Parallel()
	ctx := lifecycleCtx(t)
	testkit.AssertLifecycleIdempotent[*lifecycle]()(ctx)
}

func TestAssertLifecycleRespectsContext(t *testing.T) {
	t.Parallel()
	ctx := testkit.LifecycleContext[*lifecycle]{
		T: t,
		LifecycleBindings: testkit.LifecycleBindings[*lifecycle]{
			Factory: func() *lifecycle { l := newLifecycle(); l.opened = true; return l },
			Call: func(ctx context.Context, l *lifecycle) error {
				return l.Close(ctx)
			},
		},
	}
	testkit.AssertLifecycleRespectsContext[*lifecycle]()(ctx)
}
