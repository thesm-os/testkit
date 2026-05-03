// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
)

type itemCounter struct{ n int }

func newItemCounter(n int) *itemCounter { return &itemCounter{n: n} }

func (c *itemCounter) Count(_ context.Context) (int, error) { return c.n, nil }

func aggregatorCtx(t *testing.T, n int) testkit.AggregatorContext[*itemCounter, int] {
	t.Helper()
	return testkit.AggregatorContext[*itemCounter, int]{
		T: t,
		AggregatorBindings: testkit.AggregatorBindings[*itemCounter, int]{
			Factory: func() *itemCounter { return newItemCounter(n) },
			Call: func(ctx context.Context, c *itemCounter) (int, error) {
				return c.Count(ctx)
			},
		},
	}
}

func TestAssertAggregatorReturns(t *testing.T) {
	t.Parallel()
	ctx := aggregatorCtx(t, 42)
	testkit.AssertAggregatorReturns[*itemCounter, int](42)(ctx)
}

func TestAssertAggregatorBounded(t *testing.T) {
	t.Parallel()
	ctx := aggregatorCtx(t, 42)
	testkit.AssertAggregatorBounded[*itemCounter, int](0, 100)(ctx)
}

func TestAssertAggregatorConsistent(t *testing.T) {
	t.Parallel()
	ctx := aggregatorCtx(t, 42)
	testkit.AssertAggregatorConsistent[*itemCounter, int](5)(ctx)
}
