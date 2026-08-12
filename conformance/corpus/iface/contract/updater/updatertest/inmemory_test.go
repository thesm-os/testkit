// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package updatertest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/updater"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/updater/updatertest"
)

// updater is the model tier's under ADR-0018: `AUTO-UPDATER-REPLACES` states
// it.
//
// The suite tier still earns the pairing, because the harness seeds through the
// writer role: Get's "an error carries the zero value" check therefore runs
// against a store holding something, which is what makes the miss it asks about
// a real miss.
func TestContractContract(t *testing.T) {
	t.Parallel()

	updatertest.AssertContractContract(t,
		updatertest.ContractModel(),
		updatertest.ContractSubject("in-memory", func() updater.Contract {
			return updatertest.NewInMemory()
		}),
		updatertest.ContractOnPut("replaces rather than accumulates", func(
			tb testing.TB, subject updater.Contract, v updater.Value,
		) {
			tb.Helper()
			// The seed already wrote this key, so this is the update the
			// contract is named for.
			replacement := updater.Value{Key: v.Key, Body: v.Body + "-replaced"}
			testkit.NoError(tb, subject.Put(tb.Context(), replacement), "the update lands")

			got, err := subject.Get(tb.Context(), v.Key)
			testkit.NoError(tb, err, "and the key is still there")
			testkit.Equal(tb, got, replacement, "carrying the newer value")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	updatertest.AssertContractContract(t,
		updatertest.ContractSubject("in-memory", func() updater.Contract {
			return updatertest.NewInMemory()
		}),
		updatertest.ContractWithout("Put/smoke"),
		updatertest.ContractWithoutDouble(),
	)
}
