// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"testing"

	"go.thesmos.sh/testkit/bindings"
	"go.thesmos.sh/testkit/suite"
)

type validator struct{ valid bool }

func newValidator(v bool) *validator { return &validator{valid: v} }

func (v *validator) IsValid() bool { return v.valid }

func predicateCtx(t *testing.T, v bool) suite.PredicateContext[*validator] {
	t.Helper()
	return suite.PredicateContext[*validator]{
		T: t,
		PredicateBindings: bindings.PredicateBindings[*validator]{
			Factory: func() *validator { return newValidator(v) },
			Call:    func(val *validator) bool { return val.IsValid() },
		},
	}
}

func TestPredicate(t *testing.T) {
	t.Parallel()

	t.Run("Returns surfaces the configured value (true)", func(t *testing.T) {
		t.Parallel()
		suite.AssertPredicateReturns[*validator](true)(predicateCtx(t, true))
	})

	t.Run("Returns surfaces the configured value (false)", func(t *testing.T) {
		t.Parallel()
		suite.AssertPredicateReturns[*validator](false)(predicateCtx(t, false))
	})

	t.Run("Consistent yields equal results across N calls", func(t *testing.T) {
		t.Parallel()
		suite.AssertPredicateConsistent[*validator](5)(predicateCtx(t, true))
	})

	t.Run("RespectsContext is a structural smoke (Predicate has no ctx)", func(t *testing.T) {
		t.Parallel()
		suite.AssertPredicateRespectsContext[*validator]()(predicateCtx(t, true))
	})

	t.Run("RejectInvalid is a structural smoke (Predicate has no input)", func(t *testing.T) {
		t.Parallel()
		suite.AssertPredicateRejectInvalid[*validator]()(predicateCtx(t, true))
	})

	t.Run("ConcurrentSafe runs without races", func(t *testing.T) {
		t.Parallel()
		suite.AssertPredicateConcurrentSafe[*validator](4, 50)(predicateCtx(t, true))
	})
}
