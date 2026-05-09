// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"sync"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/bindings"
	"go.thesmos.sh/testkit/suite"
)

type accumulator struct {
	mu    sync.Mutex
	total int64
}

func newAccumulator() *accumulator { return &accumulator{} }

func (a *accumulator) Add(_ context.Context, v int64) {
	if v < 0 {
		// "invalid" sentinel input: Mutator no-ops.
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.total += v
}

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

func TestMutator(t *testing.T) {
	t.Parallel()

	t.Run("Succeeds invokes the mutator with a sample value", func(t *testing.T) {
		t.Parallel()
		suite.AssertMutatorSucceeds[*accumulator, int64](42)(mutatorCtx(t))
	})

	t.Run("Idempotent runs the mutator twice without panic", func(t *testing.T) {
		t.Parallel()
		suite.AssertMutatorIdempotent[*accumulator, int64](1)(mutatorCtx(t))
	})

	t.Run("RespectsContext does not block under cancelled ctx", func(t *testing.T) {
		t.Parallel()
		suite.AssertMutatorRespectsContext[*accumulator, int64](1)(mutatorCtx(t))
	})

	t.Run("RejectInvalid runs the consumer's check after the call", func(t *testing.T) {
		t.Parallel()
		check := func(t *testing.T, a *accumulator) {
			t.Helper()
			testkit.Equal(t, a.total, int64(0),
				"invalid input must be a no-op — total stays zero")
		}
		suite.AssertMutatorRejectInvalid[*accumulator, int64](-1, check)(mutatorCtx(t))
	})

	t.Run("RejectInvalidWith does not panic on invalid impl", func(t *testing.T) {
		t.Parallel()
		invalid := func() *accumulator { return &accumulator{} }
		suite.AssertMutatorRejectInvalidWith[*accumulator, int64](invalid, 1)(mutatorCtx(t))
	})

	t.Run("ConcurrentSafe runs without races", func(t *testing.T) {
		t.Parallel()
		suite.AssertMutatorConcurrentSafe[*accumulator, int64](1, 4, 50)(mutatorCtx(t))
	})
}
