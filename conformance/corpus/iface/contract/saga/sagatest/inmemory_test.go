// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sagatest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/saga"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/saga/sagatest"
)

// saga is the model tier's under ADR-0018: `AUTO-SAGA-FULL-COMPENSATION` states
// it, and stating it needs a sequence that fails partway — which the generated
// run arranges by stepping drawn values until one collides.
//
// Step is classified writer, so the harness seeds through it and Compensate's
// checks meet a saga with something applied. What the checks below add is the
// fingerprint's honesty: that State reflects application order, and that a
// compensated step leaves it.
func TestContractContract(t *testing.T) {
	t.Parallel()

	sagatest.AssertContractContract(t,
		sagatest.ContractModel(),
		sagatest.ContractSubject("in-memory", func() saga.Contract {
			return sagatest.NewInMemory()
		}),
		sagatest.ContractOnCompensate("undoes the step the seed applied", func(
			tb testing.TB, subject saga.Contract, v saga.Value,
		) {
			tb.Helper()
			testkit.NoError(tb, subject.Compensate(tb.Context(), v),
				"the applied step is compensated")
			testkit.ErrorIs(tb, subject.Compensate(tb.Context(), v), sagatest.ErrNotApplied,
				"and compensating it again has nothing to undo")
		}),
		sagatest.ContractOnState("fingerprints in application order", func(
			tb testing.TB, subject saga.Contract,
		) {
			tb.Helper()
			before, err := subject.State(tb.Context())
			testkit.NoError(tb, err, "the state is readable")

			first := saga.Value{Key: "b6-first", Body: "one"}
			second := saga.Value{Key: "b6-second", Body: "two"}
			testkit.NoError(tb, subject.Step(tb.Context(), first), "the first step applies")
			testkit.NoError(tb, subject.Step(tb.Context(), second), "and the second after it")

			stepped, err := subject.State(tb.Context())
			testkit.NoError(tb, err, "the state is still readable")
			testkit.NotEqual(tb, stepped, before, "two applied steps changed the fingerprint")

			testkit.NoError(tb, subject.Compensate(tb.Context(), second),
				"the newest step compensates")
			testkit.NoError(tb, subject.Compensate(tb.Context(), first),
				"then the one before it")

			after, err := subject.State(tb.Context())
			testkit.NoError(tb, err, "and the state is readable at the end")
			testkit.Equal(tb, after, before, "full compensation restored the fingerprint")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	sagatest.AssertContractContract(t,
		sagatest.ContractSubject("in-memory", func() saga.Contract {
			return sagatest.NewInMemory()
		}),
		sagatest.ContractWithout("Step/smoke"),
		sagatest.ContractWithoutDouble(),
	)
}
