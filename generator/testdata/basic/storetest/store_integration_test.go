// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storetest_test

import (
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/basic"
	"go.thesmos.sh/testkit/generator/testdata/basic/storetest"
	"go.thesmos.sh/testkit/suite"
)

// TestStoreCrossGeneratorIntegration proves the testkit's
// load-bearing architectural claim: stub, suite, and bench
// generators compose. A single fixture's interface drives all
// three generators; this test runs the suite contract against a
// stub that DelegateTo's the in-memory companion, exercising:
//
//  1. `testkit stub` output: NewStoreStub + StoreStubDelegateTo
//     forward every method call through to the in-mem.
//  2. `testkit suite` output: AssertStoreContract drives the
//     full per-shape, per-directive contract surface.
//  3. The composition: forwarded calls hit the in-mem behind the
//     stub, the in-mem returns contract-correct values, the stub
//     records call metadata and forwards the return, and the
//     contract assertions pass.
//
// If any of stub's recording, the DelegateTo wiring, or the suite's
// emission assumes things the others don't honor, this test
// surfaces it. No other fixture in the testdata exercises this
// integration — the per-generator spec / bench tests run against
// the bare in-mem.
func TestStoreCrossGeneratorIntegration(t *testing.T) {
	t.Parallel()
	AssertStoreContract(
		t,
		func() basic.Store {
			inmem := basic.NewInMemoryStore()
			inmem.Seed("test-key", basic.Item{ID: "test-id"})
			// Pass nil for the stub's tb so the auto-verification
			// at test cleanup doesn't fire — the contract assertions
			// own verification here, and binding the stub's cleanup
			// to a per-factory-invocation t would mis-order against
			// the suite's per-subtest factory calls.
			return storetest.NewStoreStub(nil, storetest.StoreStubDelegateTo(inmem))
		},
		// The stub records call metadata, which is observable state
		// distinct from the in-mem's semantic state. Atomic's
		// "failed mutation leaves state unchanged" contract uses
		// reflect.DeepEqual by default and would flag the stub's
		// recording divergence as a violation. Supply a semantic
		// comparator that inspects the stub-wrapped impl through
		// the public Get path — both impls' inner in-mems were
		// seeded identically, so a contract-correct atomic Put
		// failure leaves Get's return identical across them.
		suite.WithStateEqual(func(a, b basic.Store) bool {
			av, _ := a.Get(t.Context(), "test-key")
			bv, _ := b.Get(t.Context(), "test-key")
			return av == bv
		}),
	)
}

// BenchmarkStoreCrossGeneratorIntegration mirrors the test above
// for the bench generator. Runs `testkit bench` output against the
// stub-wrapped in-mem so any allocation cost added by the stub's
// recording surfaces in the per-method bench numbers — the contract
// is "the bench harness composes with the stub without breakage,"
// not "stubbed calls allocate zero" (they don't, by design).
func BenchmarkStoreCrossGeneratorIntegration(b *testing.B) {
	storetest.BenchmarkStoreContract(b, func() basic.Store {
		inmem := basic.NewInMemoryStore()
		inmem.Seed("test-key", basic.Item{ID: "test-id"})
		return storetest.NewStoreStub(nil, storetest.StoreStubDelegateTo(inmem))
	})
}
