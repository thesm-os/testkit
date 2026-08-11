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
// it, and stating it needs a sequence that fails partway — which a fixed run
// against one subject cannot produce.
//
// Step is classified writer, so the harness seeds through it and Compensate's
// checks meet a saga with something applied. A subject that panicked on a
// compensation, or applied one for a cancelled caller, fails before any law
// runs.
func TestContractContract(t *testing.T) {
	t.Parallel()

	sagatest.AssertContractContract(t,
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
