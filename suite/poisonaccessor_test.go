// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"testing"

	"go.thesmos.sh/testkit/bindings"
	"go.thesmos.sh/testkit/suite"
)

type healthChecker struct{ err error }

func newHealthChecker() *healthChecker { return &healthChecker{} }

func (h *healthChecker) Err() error { return h.err }

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

func TestAssertPoisonAccessorNilOnFresh(t *testing.T) {
	t.Parallel()
	ctx := poisonAccessorCtx(t)
	suite.AssertPoisonAccessorNilOnFresh[*healthChecker]()(ctx)
}

func TestAssertPoisonAccessorConsistent(t *testing.T) {
	t.Parallel()
	ctx := poisonAccessorCtx(t)
	suite.AssertPoisonAccessorConsistent[*healthChecker]()(ctx)
}
