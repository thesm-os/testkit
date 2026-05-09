// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/basic"
)

// TestStoreContract closes the loop on `testkit suite`: the
// generated [AssertStoreContract] driver runs against a real impl,
// proving the rendered template links against the suite runtime
// with the right type args and that the per-shape baselines
// actually pass when the impl is contract-correct.
//
// The factory pre-seeds "test-key" → Item{ID: "test-id"} because
// the generated baseline asserts Get("test-key") == Item{ID:
// "test-id"} — sample-driven literals are part of the contract.
func TestStoreContract(t *testing.T) {
	t.Parallel()
	AssertStoreContract(t, seededStoreFactory)
}

// TestStoreContractAcrossImpls exercises the generated
// [AssertStoreContractAcrossImpls] entry point — the multi-impl
// scaffold a consumer uses to run one contract across N
// implementations under per-impl t.Run subtests. We supply two
// factories that build the same in-mem differently (direct Seed
// vs. Seed-plus-Put-of-an-extra-item) so the generated
// `t.Run(name, ...)` scaffold is exercised end-to-end and any
// divergence between [AssertStoreContract] and AcrossImpls
// dispatch surfaces here.
func TestStoreContractAcrossImpls(t *testing.T) {
	t.Parallel()
	AssertStoreContractAcrossImpls(t, []StoreNamedFactory{
		{Name: "Seeded", Factory: seededStoreFactory},
		{Name: "PutPlusSeeded", Factory: putPlusSeededStoreFactory},
	})
}

// seededStoreFactory builds an InMemoryStore with the contract's
// sample (key, value) seeded directly via Seed.
func seededStoreFactory() basic.Store {
	s := basic.NewInMemoryStore()
	s.Seed("test-key", basic.Item{ID: "test-id"})
	return s
}

// putPlusSeededStoreFactory installs the contract's sample value
// via Seed (so the Reader baseline lands on it) AND also Puts an
// extra item via the public Put path. Both factories produce
// contract-correct impls; AcrossImpls runs the full Store contract
// against each.
func putPlusSeededStoreFactory() basic.Store {
	s := basic.NewInMemoryStore()
	s.Seed("test-key", basic.Item{ID: "test-id"})
	_ = s.Put(context.Background(), basic.Item{ID: "another-id", Name: "another"})
	return s
}
