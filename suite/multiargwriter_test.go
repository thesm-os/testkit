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
			"k", "v", 1)(multiArgCtx(t))
	})

	t.Run("WriteRejectInvalid surfaces the sentinel", func(t *testing.T) {
		t.Parallel()
		suite.AssertMultiArgWriteRejectInvalid[*scheduler, string, string, int](
			"k", "v", -1, errSchedInvalid)(multiArgCtx(t))
	})

	t.Run("Idempotent succeeds on repeat write", func(t *testing.T) {
		t.Parallel()
		suite.AssertMultiArgWriterIdempotent[*scheduler, string, string, int](
			"k", "v", 1)(multiArgCtx(t))
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
			"k", "v", 1)(ctx)
	})

	t.Run("ConcurrentSafe runs without races", func(t *testing.T) {
		t.Parallel()
		suite.AssertMultiArgWriterConcurrentSafe[*scheduler, string, string, int](
			"k", "v", 1, 4, 50)(multiArgCtx(t))
	})
}
