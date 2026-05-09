// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/testkit/bindings"
	"go.thesmos.sh/testkit/suite"
)

type healthChecker struct{ err error }

func newHealthChecker() *healthChecker { return &healthChecker{} }

func (h *healthChecker) Err() error { return h.err }

var errPoisoned = errors.New("poisoned")

func poisonAccessorCtx(t *testing.T) suite.PoisonAccessorContext[*healthChecker] {
	t.Helper()
	return suite.PoisonAccessorContext[*healthChecker]{
		T: t,
		PoisonAccessorBindings: bindings.PoisonAccessorBindings[*healthChecker]{
			Factory: newHealthChecker,
			Call:    func(h *healthChecker) error { return h.Err() },
		},
	}
}

func TestPoisonAccessor(t *testing.T) {
	t.Parallel()

	t.Run("NilOnFresh returns nil for a fresh impl", func(t *testing.T) {
		t.Parallel()
		suite.AssertPoisonAccessorNilOnFresh[*healthChecker]()(poisonAccessorCtx(t))
	})

	t.Run("Consistent yields the same nil/non-nil across calls", func(t *testing.T) {
		t.Parallel()
		suite.AssertPoisonAccessorConsistent[*healthChecker]()(poisonAccessorCtx(t))
	})

	t.Run("RespectsContext is a structural smoke (PoisonAccessor has no ctx)", func(t *testing.T) {
		t.Parallel()
		suite.AssertPoisonAccessorRespectsContext[*healthChecker]()(poisonAccessorCtx(t))
	})

	t.Run("RejectInvalid surfaces the sentinel against a poisoned factory", func(t *testing.T) {
		t.Parallel()
		poisoned := func() *healthChecker { return &healthChecker{err: errPoisoned} }
		suite.AssertPoisonAccessorRejectInvalid[*healthChecker](poisoned, errPoisoned)(
			poisonAccessorCtx(t))
	})

	t.Run("ConcurrentSafe runs without races", func(t *testing.T) {
		t.Parallel()
		suite.AssertPoisonAccessorConcurrentSafe[*healthChecker](4, 50)(poisonAccessorCtx(t))
	})
}

func TestAssertPoisonAccessorBaseline(t *testing.T) {
	t.Parallel()
	suite.AssertPoisonAccessorBaseline[*healthChecker]()(poisonAccessorCtx(t))
}
