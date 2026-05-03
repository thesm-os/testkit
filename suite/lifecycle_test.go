// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"errors"
	"testing"

	"go.thesmos.sh/testkit/bindings"
	"go.thesmos.sh/testkit/suite"
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

func TestAssertLifecycleSucceeds(t *testing.T) {
	t.Parallel()
	ctx := lifecycleCtx(t)
	suite.AssertLifecycleSucceeds[*lifecycle]()(ctx)
}

func TestAssertLifecycleIdempotent(t *testing.T) {
	t.Parallel()
	ctx := lifecycleCtx(t)
	suite.AssertLifecycleIdempotent[*lifecycle]()(ctx)
}

func TestAssertLifecycleRespectsContext(t *testing.T) {
	t.Parallel()
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
}
