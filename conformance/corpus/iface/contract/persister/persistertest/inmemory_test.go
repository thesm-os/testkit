// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package persistertest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/persister"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/persister/persistertest"
)

// persister is the model tier's under ADR-0018: `AUTO-PERSISTER-RETRIEVABLE`
// states it, and it needs a reference implementation to compare against.
//
// The suite tier still earns the pairing. Put is classified writer, so the
// harness seeds through it and Get's "an error carries the zero value" check
// runs against a store that holds something — which is what makes the miss it
// asks about a real miss rather than an empty store answering nothing.
func TestContractContract(t *testing.T) {
	t.Parallel()

	persistertest.AssertContractContract(t,
		persistertest.ContractModel(),
		persistertest.ContractSubject("in-memory", func() persister.Contract {
			return persistertest.NewInMemory()
		}),
		persistertest.ContractOnGet("returns what the seed wrote", func(
			tb testing.TB, subject persister.Contract, key string,
		) {
			tb.Helper()
			got, err := subject.Get(tb.Context(), key)
			testkit.NoError(tb, err, "the seeded key is found")
			testkit.Equal(tb, got.Key, key, "carrying the key it was filed under")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	persistertest.AssertContractContract(t,
		persistertest.ContractSubject("in-memory", func() persister.Contract {
			return persistertest.NewInMemory()
		}),
		persistertest.ContractWithout("Put/smoke"),
		persistertest.ContractWithoutDouble(),
	)
}
