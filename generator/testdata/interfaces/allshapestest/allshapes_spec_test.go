// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package allshapestest_test

import (
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/interfaces"
	"go.thesmos.sh/testkit/suite"
)

// TestAllShapesContract closes the e2e loop on the suite generator's
// 21-shape coverage. The seed function configures
// [interfaces.InMemoryAllShapes] so every per-method baseline lands on
// the contract's sample literals — for state-derived methods (Reader,
// Lookup, BatchReader) by pre-populating items[<sample-key>] = <sample-value>,
// and for aggregate / pure / stream-consumer methods via the in-mem's
// contract-alignment overrides.
//
// The samples here mirror what the generator's default sample
// renderer emits per shape:
//   - "test-key" / "test-id" for Reader-shape (V, K) slots
//   - "test-result0" / "test-result1" for unnamed multi-result string
//     slots (Lookup secondary, MultiReader V2, Pure return)
//   - 42 / 43 for unnamed multi-result int slots (Aggregator,
//     MultiAggregator V1, MultiAggregator V2, StreamConsumer)
//
// The InvalidFactory option supplies a separate in-mem in invalidMode
// so AssertLifecycleRejectInvalidWith / AssertVoidLifecycleRejectInvalidWith
// / AssertMutatorRejectInvalidWith fire — each verifies a contract
// against a misconfigured impl. When the suite generator's sample
// defaults change, the values configured here change alongside it.
func TestAllShapesContract(t *testing.T) {
	t.Parallel()
	AssertAllShapesContract(t, func() interfaces.AllShapes {
		s := interfaces.NewInMemoryAllShapes()
		// Seed items["test-key"] = Record{ID:"test-id"} so Reader /
		// Lookup / ReaderNoError / ReaderWithBool / PointerReader /
		// MultiReader / Lookup / BatchReader baselines land on the
		// sample (key, value) pair the contract asserts.
		s.SeedAt("test-key", interfaces.Record{ID: "test-id"})
		// Pure baseline expects Description() to equal "test-result0"
		// (SampleResultAt(0) for an unnamed string return).
		s.SetDescription("test-result0")
		// Aggregator/MultiAggregator/StreamConsumer baselines expect
		// the sampled int values.
		s.SetCountOverride(42)
		s.SetStatsOverride(42, 43)
		s.SetReadFromOverride(42)
		// Lookup secondary R and MultiReader V2 baselines expect
		// "test-result1" (SampleResultAt(1) for an unnamed string).
		s.SetInspectMeta("test-result1")
		s.SetFetchMeta("test-result1")
		return s
	}, suite.WithInvalidFactory(func() interfaces.AllShapes {
		// The "invalid" impl flips invalidMode so Init returns
		// ErrNotFound — the contract for Lifecycle's
		// rejects-invalid extra requires err != nil from a
		// misconfigured impl. Touch / Reset under invalidMode no-op
		// without panic, satisfying the no-panic guarantee that
		// AssertMutatorRejectInvalidWith and
		// AssertVoidLifecycleRejectInvalidWith verify.
		s := interfaces.NewInMemoryAllShapes()
		s.SetInvalidMode(true)
		return s
	}))
}
