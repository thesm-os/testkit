// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/bindings"
	"go.thesmos.sh/testkit/suite"
)

type itemCounter struct{ n int }

func newItemCounter(n int) *itemCounter { return &itemCounter{n: n} }

func (c *itemCounter) Count(_ context.Context) (int, error) { return c.n, nil }

func aggregatorCtx(t *testing.T, n int) suite.AggregatorContext[*itemCounter, int] {
	t.Helper()
	return suite.AggregatorContext[*itemCounter, int]{
		T: t,
		AggregatorBindings: bindings.AggregatorBindings[*itemCounter, int]{
			Factory: func() *itemCounter { return newItemCounter(n) },
			Call: func(ctx context.Context, c *itemCounter) (int, error) {
				return c.Count(ctx)
			},
		},
	}
}

func TestAggregator(t *testing.T) {
	t.Parallel()

	t.Run("Returns surfaces the configured value", func(t *testing.T) {
		t.Parallel()
		suite.AssertAggregatorReturns[*itemCounter, int](42)(aggregatorCtx(t, 42))
	})

	t.Run("Bounded asserts the value falls within the range", func(t *testing.T) {
		t.Parallel()
		suite.AssertAggregatorBounded[*itemCounter, int](0, 100)(aggregatorCtx(t, 42))
	})

	t.Run("Consistent yields equal results across N calls", func(t *testing.T) {
		t.Parallel()
		suite.AssertAggregatorConsistent[*itemCounter, int](5)(aggregatorCtx(t, 42))
	})

	t.Run("RespectsContext surfaces ctx.Canceled on cancelled call", func(t *testing.T) {
		t.Parallel()
		ctx := suite.AggregatorContext[*itemCounter, int]{
			T: t,
			AggregatorBindings: bindings.AggregatorBindings[*itemCounter, int]{
				Factory: func() *itemCounter { return newItemCounter(0) },
				Call: func(c context.Context, _ *itemCounter) (int, error) {
					return 0, c.Err()
				},
			},
		}
		suite.AssertAggregatorRespectsContext[*itemCounter, int]()(ctx)
	})

	t.Run("ConcurrentSafe runs without races", func(t *testing.T) {
		t.Parallel()
		suite.AssertAggregatorConcurrentSafe[*itemCounter, int](4, 50)(aggregatorCtx(t, 42))
	})
}
