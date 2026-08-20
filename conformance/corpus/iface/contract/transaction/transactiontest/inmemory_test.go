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

// Run takes a unit of work, which no literal can be written for, so every
// check the rules reached for it was refused — the header lists both. What
// the contract actually claims lives in the row below, which supplies the
// body itself.
func TestContractContract(t *testing.T) {
	t.Parallel()

	transactiontest.RunContract(t,
		transactiontest.ContractHarness[*transactiontest.InMemory]{Name: "in-memory", New: transactiontest.NewInMemory},
		transactiontest.ContractChecks{
			{
				Method: "Get",
				Name:   "erroring-body-changes-nothing",
				Claim:  "an erroring body leaves the store as it found it",
				Run: func(tb testing.TB, s transaction.Contract, fx transactiontest.ContractFixture) {
					tb.Helper()
					before, beforeErr := s.Get(tb.Context(), transactiontest.RunKey)

					induced := errors.New("transactiontest_test: induced")
					testkit.ErrorIs(tb,
						s.Run(tb.Context(), func(context.Context) error { return induced }),
						induced, "the body's error is the run's")

					after, afterErr := s.Get(tb.Context(), transactiontest.RunKey)
					testkit.Equal(tb, afterErr == nil, beforeErr == nil,
						"the erroring run changed no presence")
					testkit.Equal(tb, after, before, "and no value")

					testkit.NoError(tb, s.Run(tb.Context(), nil),
						"an empty unit of work commits")
					_, err := s.Get(tb.Context(), transactiontest.RunKey)
					testkit.NoError(tb, err, "and its entry is readable")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	transactiontest.RunContract(t,
		transactiontest.ContractHarness[*transactiontest.InMemory]{Name: "in-memory", New: transactiontest.NewInMemory},
		transactiontest.ContractSuite.Without(transactiontest.ContractSuite.Checks.Put.Smoke()),
	)
}
