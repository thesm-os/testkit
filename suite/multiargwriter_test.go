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

type scheduler struct {
	mu      sync.Mutex
	written int
}

var errSchedInvalid = errors.New("schedule: invalid")

func (s *scheduler) Schedule(_ context.Context, key, value string, priority int) error {
	if priority < 0 {
		return errSchedInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.written++
	_, _, _ = key, value, priority
	return nil
}

func multiArgCtx(t *testing.T) suite.MultiArgWriterContext[*scheduler, string, string, int] {
	t.Helper()
	return suite.MultiArgWriterContext[*scheduler, string, string, int]{
		T: t,
		MultiArgWriterBindings: bindings.MultiArgWriterBindings[*scheduler, string, string, int]{
			Factory: func() *scheduler { return &scheduler{} },
			Call: func(ctx context.Context, s *scheduler, p1, p2 string, p3 int) error {
				return s.Schedule(ctx, p1, p2, p3)
			},
		},
	}
}

func TestMultiArgWriter(t *testing.T) {
	t.Parallel()

	t.Run("WriteSucceeds for valid args", func(t *testing.T) {
		t.Parallel()
		suite.AssertMultiArgWriteSucceeds[*scheduler, string, string, int](
			"k", "v", 1,
		)(multiArgCtx(t))
	})

	t.Run("WriteRejectInvalid surfaces the sentinel", func(t *testing.T) {
		t.Parallel()
		suite.AssertMultiArgWriteRejectInvalid[*scheduler, string, string, int](
			"k", "v", -1, errSchedInvalid,
		)(multiArgCtx(t))
	})

	t.Run("Idempotent succeeds on repeat write", func(t *testing.T) {
		t.Parallel()
		suite.AssertMultiArgWriterIdempotent[*scheduler, string, string, int](
			"k", "v", 1,
		)(multiArgCtx(t))
	})

	t.Run("RespectsContext surfaces ctx.Canceled", func(t *testing.T) {
		t.Parallel()
		ctx := suite.MultiArgWriterContext[*scheduler, string, string, int]{
			T: t,
			MultiArgWriterBindings: bindings.MultiArgWriterBindings[*scheduler, string, string, int]{
				Factory: func() *scheduler { return &scheduler{} },
				Call: func(c context.Context, _ *scheduler, _, _ string, _ int) error {
					return c.Err()
				},
			},
		}
		suite.AssertMultiArgWriterRespectsContext[*scheduler, string, string, int](
			"k", "v", 1,
		)(ctx)
	})

	t.Run("ConcurrentSafe runs without races", func(t *testing.T) {
		t.Parallel()
		suite.AssertMultiArgWriterConcurrentSafe[*scheduler, string, string, int](
			"k", "v", 1, 4, 50,
		)(multiArgCtx(t))
	})
}

func multiArgVariadicCtx(t *testing.T) suite.MultiArgWriterVariadicContext[*scheduler] {
	t.Helper()
	return suite.MultiArgWriterVariadicContext[*scheduler]{
		T: t,
		MultiArgWriterVariadicBindings: bindings.MultiArgWriterVariadicBindings[*scheduler]{
			Factory: func() *scheduler { return &scheduler{} },
			Call: func(ctx context.Context, s *scheduler, args ...any) error {
				key, _ := args[0].(string)
				value, _ := args[1].(string)
				priority, _ := args[2].(int)
				return s.Schedule(ctx, key, value, priority)
			},
		},
	}
}

func TestMultiArgWriterVariadic(t *testing.T) {
	t.Parallel()

	t.Run("WriteSucceeds for valid args", func(t *testing.T) {
		t.Parallel()
		suite.AssertMultiArgWriteSucceedsVariadic[*scheduler](
			"k", "v", 1,
		)(multiArgVariadicCtx(t))
	})

	t.Run("WriteRejectInvalid surfaces the sentinel", func(t *testing.T) {
		t.Parallel()
		suite.AssertMultiArgWriteRejectInvalidVariadic[*scheduler](
			[]any{"k", "v", -1}, errSchedInvalid,
		)(multiArgVariadicCtx(t))
	})

	t.Run("Idempotent succeeds on repeat write", func(t *testing.T) {
		t.Parallel()
		suite.AssertMultiArgWriterIdempotentVariadic[*scheduler](
			"k", "v", 1,
		)(multiArgVariadicCtx(t))
	})

	t.Run("RespectsContext surfaces ctx.Canceled", func(t *testing.T) {
		t.Parallel()
		ctx := suite.MultiArgWriterVariadicContext[*scheduler]{
			T: t,
			MultiArgWriterVariadicBindings: bindings.MultiArgWriterVariadicBindings[*scheduler]{
				Factory: func() *scheduler { return &scheduler{} },
				Call: func(c context.Context, _ *scheduler, _ ...any) error {
					return c.Err()
				},
			},
		}
		suite.AssertMultiArgWriterRespectsContextVariadic[*scheduler](
			"k", "v", 1,
		)(ctx)
	})

	t.Run("ConcurrentSafe runs without races", func(t *testing.T) {
		t.Parallel()
		suite.AssertMultiArgWriterConcurrentSafeVariadic[*scheduler](
			4, 50, "k", "v", 1,
		)(multiArgVariadicCtx(t))
	})
}

func TestAssertMultiArgWriterBaselineVariadic(t *testing.T) {
	t.Parallel()
	ctx := suite.MultiArgWriterVariadicContext[*scheduler]{
		T: t,
		MultiArgWriterVariadicBindings: bindings.MultiArgWriterVariadicBindings[*scheduler]{
			Factory: func() *scheduler { return &scheduler{} },
			Call: func(c context.Context, s *scheduler, args ...any) error {
				if err := c.Err(); err != nil {
					return err
				}
				key, _ := args[0].(string)
				value, _ := args[1].(string)
				priority, _ := args[2].(int)
				return s.Schedule(c, key, value, priority)
			},
		},
	}
	suite.AssertMultiArgWriterBaselineVariadic[*scheduler](
		[]any{"k", "v", 1},
	)(ctx)
}
