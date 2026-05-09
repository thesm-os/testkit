// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directivestest_test

import (
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/interfaces"
	"go.thesmos.sh/testkit/generator/testdata/interfaces/directivestest"
)

// BenchmarkDirectives closes the loop on `testkit bench` for an
// interface stress-testing every supported directive (`errors`,
// `wrapped-via`, `deprecated`, `retryable`, `retry-succeeds-on-attempt`,
// `order-after`, `partition`, `atomic`, `idempotent`,
// `integration-only`).
//
// Bench primitives don't assert on return values — they just
// measure call cost — so directive-driven impls that always error
// (e.g. Wrap returning an ErrInternal-wrapped error) still produce
// meaningful timings.
func BenchmarkDirectives(b *testing.B) {
	directivestest.BenchmarkDirectivesContract(b, func() interfaces.Directives {
		return interfaces.NewInMemoryDirectives()
	})
}
