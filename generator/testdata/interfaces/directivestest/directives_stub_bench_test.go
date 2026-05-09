// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directivestest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/interfaces"
	"go.thesmos.sh/testkit/generator/testdata/interfaces/directivestest"
)

// BenchmarkDirectivesStub runs the contract driver against the
// generated stub in BenchMode for every directive-stress-tested
// method (errors, wrapped-via, retryable, deprecated, partition,
// order-after). With recording disabled and no-op Func overrides,
// the dispatch must be alloc-free even when directive hooks (fault
// chain, retry counter, order-after assertion) wire into the path.
func BenchmarkDirectivesStub(b *testing.B) {
	directivestest.BenchmarkDirectivesContract(b, func() interfaces.Directives {
		return directivestest.NewDirectivesStub(b,
			directivestest.DirectivesStubBenchMode(),
			directivestest.WithDirectivesLegacy(func(_ context.Context, _ interfaces.Record) error { return nil }),
			directivestest.WithDirectivesOpen(func(_ context.Context) error { return nil }),
			directivestest.WithDirectivesRead(func(_ context.Context, _ string) (interfaces.Record, error) {
				return interfaces.Record{}, nil
			}),
			directivestest.WithDirectivesRetry(func(_ context.Context, _ string) error { return nil }),
			directivestest.WithDirectivesShard(func(_ context.Context, _ interfaces.Record) error { return nil }),
			directivestest.WithDirectivesShardByKey(func(_ context.Context, _ string) error { return nil }),
			directivestest.WithDirectivesSubmit(func(_ context.Context, _ interfaces.Record) error { return nil }),
			directivestest.WithDirectivesWrap(func(_ context.Context, _ string) error { return nil }),
		)
	})
}
