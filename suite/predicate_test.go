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

func TestAssertPredicateConsistent(t *testing.T) {
	t.Parallel()
	ctx := predicateCtx(t, true)
	suite.AssertPredicateConsistent[*validator](5)(ctx)
}

func TestAssertPredicateReturns(t *testing.T) {
	t.Parallel()

	t.Run("true", func(t *testing.T) {
		t.Parallel()
		ctx := predicateCtx(t, true)
		suite.AssertPredicateReturns[*validator](true)(ctx)
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()
		ctx := predicateCtx(t, false)
		suite.AssertPredicateReturns[*validator](false)(ctx)
	})
}
