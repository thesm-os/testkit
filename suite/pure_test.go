// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"testing"

	"go.thesmos.sh/testkit/bindings"
	"go.thesmos.sh/testkit/suite"
)

type counter struct{ n int }

func newCounter() *counter { return &counter{n: 42} }

func (c *counter) Value() int { return c.n }

func pureCtx(t *testing.T) suite.PureContext[*counter, int] {
	t.Helper()
	return suite.PureContext[*counter, int]{
		T: t,
		PureBindings: bindings.PureBindings[*counter, int]{
			Factory: newCounter,
			Call:    func(c *counter) int { return c.Value() },
		},
	}
}

func TestPure(t *testing.T) {
	t.Parallel()

	t.Run("Returns surfaces the configured value", func(t *testing.T) {
		t.Parallel()
		suite.AssertPureReturns[*counter, int](42)(pureCtx(t))
	})

	t.Run("Deterministic yields equal results across N calls", func(t *testing.T) {
		t.Parallel()
		suite.AssertDeterministic[*counter, int](5)(pureCtx(t))
	})

	t.Run("NoSideEffects observes no state change across the call", func(t *testing.T) {
		t.Parallel()
		suite.AssertNoSideEffects[*counter, int, int](
			func(c *counter) int { return c.n },
		)(pureCtx(t))
	})

	t.Run("RespectsContext is a structural smoke (Pure has no ctx)", func(t *testing.T) {
		t.Parallel()
		suite.AssertPureRespectsContext[*counter, int]()(pureCtx(t))
	})

	t.Run("RejectInvalid is a structural smoke (Pure has no input)", func(t *testing.T) {
		t.Parallel()
		suite.AssertPureRejectInvalid[*counter, int]()(pureCtx(t))
	})

	t.Run("ConcurrentSafe runs without races", func(t *testing.T) {
		t.Parallel()
		suite.AssertPureConcurrentSafe[*counter, int](4, 50)(pureCtx(t))
	})
}
