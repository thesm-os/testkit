// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"testing"

	"go.thesmos.sh/testkit/bench"
	"go.thesmos.sh/testkit/bindings"
)

func benchPoisonAccessorCtx(b *testing.B) bench.PoisonAccessorContext[*healthChecker] {
	b.Helper()
	return bench.PoisonAccessorContext[*healthChecker]{
		B: b,
		PoisonAccessorBindings: bindings.PoisonAccessorBindings[*healthChecker]{
			Factory: newHealthChecker,
			Call:    func(h *healthChecker) error { return h.Err() },
		},
	}
}

func BenchmarkPoisonAccessorAllocsWithin(b *testing.B) {
	ctx := benchPoisonAccessorCtx(b)
	bench.PoisonAccessorAllocsWithin[*healthChecker](0)(ctx)
}
