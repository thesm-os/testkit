// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/bindings"
	"go.thesmos.sh/testkit/suite"
)

type accumulator struct{ total int64 }

func newAccumulator() *accumulator { return &accumulator{} }

func (a *accumulator) Add(_ context.Context, v int64) { a.total += v }

func mutatorCtx(t *testing.T) suite.MutatorContext[*accumulator, int64] {
	t.Helper()
	return suite.MutatorContext[*accumulator, int64]{
		T: t,
		MutatorBindings: bindings.MutatorBindings[*accumulator, int64]{
			Factory: newAccumulator,
			Call: func(ctx context.Context, a *accumulator, v int64) {
				a.Add(ctx, v)
			},
		},
	}
}

func TestAssertMutatorSucceeds(t *testing.T) {
	t.Parallel()
	ctx := mutatorCtx(t)
	suite.AssertMutatorSucceeds[*accumulator, int64](42)(ctx)
}

func TestAssertMutatorIdempotent(t *testing.T) {
	t.Parallel()
	ctx := mutatorCtx(t)
	suite.AssertMutatorIdempotent[*accumulator, int64](1)(ctx)
}
