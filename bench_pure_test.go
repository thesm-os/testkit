// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"testing"

	"go.thesmos.sh/testkit"
)

func benchPureCtx(b *testing.B) testkit.BenchPureContext[*counter, int] {
	b.Helper()
	return testkit.BenchPureContext[*counter, int]{
		B: b,
		PureBindings: testkit.PureBindings[*counter, int]{
			Factory: newCounter,
			Call:    func(c *counter) int { return c.Value() },
		},
	}
}

func BenchmarkPureAllocsWithin(b *testing.B) {
	ctx := benchPureCtx(b)
	testkit.BenchPureAllocsWithin[*counter, int](0)(ctx)
}
