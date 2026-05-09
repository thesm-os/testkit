// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package allshapestest_test

import (
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/interfaces"
	"go.thesmos.sh/testkit/generator/testdata/interfaces/allshapestest"
)

// BenchmarkAllShapes closes the loop on `testkit bench` for an
// interface that exercises every signature-tier shape (Reader,
// Writer, Mutator, Lifecycle, Pure, Predicate, BatchReader,
// CompositeWriter, MultiAggregator, MultiReader, PointerReader,
// PoisonAccessor, ReaderNoError, ReaderWithBool, StreamReader,
// StreamConsumer, VoidLifecycle, Aggregator, Lookup, Deleter, …).
//
// Unlike the suite contract, bench primitives don't assert on
// return values — they only measure ns/op and allocs/op. So the
// in-mem can return ErrNotFound for unseeded keys without failing
// the benchmark; the timings simply reflect the not-found path.
func BenchmarkAllShapes(b *testing.B) {
	allshapestest.BenchmarkAllShapesContract(b, func() interfaces.AllShapes {
		return interfaces.NewInMemoryAllShapes()
	})
}
