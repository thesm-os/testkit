// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package upsertertest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/upserter"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/upserter/upsertertest"
)

// upserter is the model tier's under ADR-0018: `AUTO-UPSERTER-IDEMPOTENT`
// states it, and stating it needs an observation of the whole store rather than
// of one read.
//
// The suite tier gets the pairing the seed produces: Get's "an error carries
// the zero value" check runs against a store holding something, which is what
// makes the miss it asks about a real miss.
func TestContractContract(t *testing.T) {
	t.Parallel()

	upsertertest.AssertContractContract(t,
		upsertertest.ContractModel(),
		upsertertest.ContractSubject("in-memory", func() upserter.Contract {
			return upsertertest.NewInMemory()
		}),
		upsertertest.ContractOnPut("writes the same key the seed did", func(
			tb testing.TB, subject upserter.Contract, v upserter.Value,
		) {
			tb.Helper()
			// The seed already wrote this value, so the call under check is the
			// repeat the contract is named for.
			testkit.NoError(tb, subject.Put(tb.Context(), v), "the repeated write lands")

			got, err := subject.Get(tb.Context(), v.Key)
			testkit.NoError(tb, err, "and the key is still there")
			testkit.Equal(tb, got, v, "carrying what it carried before")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	upsertertest.AssertContractContract(t,
		upsertertest.ContractSubject("in-memory", func() upserter.Contract {
			return upsertertest.NewInMemory()
		}),
		upsertertest.ContractWithout("Put/smoke"),
		upsertertest.ContractWithoutDouble(),
	)
}

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestContractSaturation(t *testing.T) {
	t.Parallel()
	upsertertest.ContractModelSaturation(t, func() upserter.Contract {
		return upsertertest.NewInMemory()
	})
}
