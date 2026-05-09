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

type stats struct {
	count, total int
	failNext     bool
}

var errStatsUnavailable = errors.New("stats: unavailable")

func newStats(count, total int) *stats { return &stats{count: count, total: total} }

func (s *stats) Stats(_ context.Context) (int, int, error) {
	if s.failNext {
		return 0, 0, errStatsUnavailable
	}
	return s.count, s.total, nil
}

func multiAggregatorCtx(t *testing.T, count, total int) suite.MultiAggregatorContext[*stats, int, int] {
	t.Helper()
	return suite.MultiAggregatorContext[*stats, int, int]{
		T: t,
		MultiAggregatorBindings: bindings.MultiAggregatorBindings[*stats, int, int]{
			Factory: func() *stats { return newStats(count, total) },
			Call: func(ctx context.Context, s *stats) (int, int, error) {
				return s.Stats(ctx)
			},
		},
	}
}

func TestMultiAggregator(t *testing.T) {
	t.Parallel()

	t.Run("Returns surfaces both values", func(t *testing.T) {
		t.Parallel()
		suite.AssertMultiAggregatorReturns[*stats, int, int](3, 42)(
			multiAggregatorCtx(t, 3, 42))
	})

	t.Run("ReturnsSentinel surfaces the configured error against an invalid factory", func(t *testing.T) {
		t.Parallel()
		invalidFactory := func() *stats { return &stats{failNext: true} }
		suite.AssertMultiAggregatorReturnsSentinel[*stats, int, int](
			invalidFactory, errStatsUnavailable)(multiAggregatorCtx(t, 0, 0))
	})

	t.Run("Consistent yields equal values across N calls", func(t *testing.T) {
		t.Parallel()
		suite.AssertMultiAggregatorConsistent[*stats, int, int](4)(
			multiAggregatorCtx(t, 3, 42))
	})

	t.Run("RespectsContext surfaces ctx.Canceled", func(t *testing.T) {
		t.Parallel()
		ctx := suite.MultiAggregatorContext[*stats, int, int]{
			T: t,
			MultiAggregatorBindings: bindings.MultiAggregatorBindings[*stats, int, int]{
				Factory: func() *stats { return newStats(0, 0) },
				Call: func(c context.Context, _ *stats) (int, int, error) {
					return 0, 0, c.Err()
				},
			},
		}
		suite.AssertMultiAggregatorRespectsContext[*stats, int, int]()(ctx)
	})

	t.Run("ConcurrentSafe runs without races", func(t *testing.T) {
		t.Parallel()
		suite.AssertMultiAggregatorConcurrentSafe[*stats, int, int](4, 50)(
			multiAggregatorCtx(t, 3, 42))
	})
}

func TestAssertMultiAggregatorBaseline(t *testing.T) {
	t.Parallel()
	ctx := suite.MultiAggregatorContext[*stats, int, int]{
		T: t,
		MultiAggregatorBindings: bindings.MultiAggregatorBindings[*stats, int, int]{
			Factory: func() *stats { return newStats(0, 0) },
			Call: func(c context.Context, s *stats) (int, int, error) {
				if err := c.Err(); err != nil {
					return 0, 0, err
				}
				return s.Stats(c)
			},
		},
	}
	suite.AssertMultiAggregatorBaseline[*stats, int, int](0, 0)(ctx)
}
