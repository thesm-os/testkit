// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package transactiontest_test

import (
	"context"
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/transaction"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/transaction/transactiontest"
)

// transaction is the model tier's under ADR-0018: `AUTO-TRANSACTION-ROLLBACK`
// states it, inducing the failure through the body Run now accepts.
//
// The check below is the deterministic complement: one erroring body, one
// committing one, and the read-back in between — the exact sequence the law
// draws its way to.
func TestContractContract(t *testing.T) {
	t.Parallel()

	transactiontest.AssertContractContract(t,
		transactiontest.ContractModel(),
		transactiontest.ContractSubject("in-memory", func() transaction.Contract {
			return transactiontest.NewInMemory()
		}),
		transactiontest.ContractOnGet("an erroring body leaves the store as it found it", func(
			tb testing.TB, subject transaction.Contract, _ string,
		) {
			tb.Helper()
			before, beforeErr := subject.Get(tb.Context(), transactiontest.RunKey)

			induced := errors.New("transactiontest_test: induced")
			testkit.ErrorIs(tb,
				subject.Run(tb.Context(), func(context.Context) error { return induced }),
				induced, "the body's error is the run's")

			after, afterErr := subject.Get(tb.Context(), transactiontest.RunKey)
			testkit.Equal(tb, afterErr == nil, beforeErr == nil,
				"the erroring run changed no presence")
			testkit.Equal(tb, after, before, "and no value")

			testkit.NoError(tb, subject.Run(tb.Context(), nil),
				"an empty unit of work commits")
			_, err := subject.Get(tb.Context(), transactiontest.RunKey)
			testkit.NoError(tb, err, "and its entry is readable")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	transactiontest.AssertContractContract(t,
		transactiontest.ContractSubject("in-memory", func() transaction.Contract {
			return transactiontest.NewInMemory()
		}),
		transactiontest.ContractWithout("Run/smoke"),
		transactiontest.ContractWithoutDouble(),
	)
}

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestContractSaturation(t *testing.T) {
	t.Parallel()
	transactiontest.ContractModelSaturation(t, func() transaction.Contract {
		return transactiontest.NewInMemory()
	})
}
