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

func TestAssertDeterministic(t *testing.T) {
	t.Parallel()
	ctx := pureCtx(t)
	suite.AssertDeterministic[*counter, int](5)(ctx)
}

func TestAssertNoSideEffects(t *testing.T) {
	t.Parallel()
	ctx := pureCtx(t)
	suite.AssertNoSideEffects[*counter, int, int](
		func(c *counter) int { return c.n },
	)(ctx)
}
