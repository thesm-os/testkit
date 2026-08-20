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
// it, and stating it needs a sequence that fails partway.
//
// What the rows add is the fingerprint's honesty: that State reflects
// application order, and that a compensated step leaves it.
func TestContractContract(t *testing.T) {
	t.Parallel()

	sagatest.RunContract(t,
		sagatest.ContractHarness[*sagatest.InMemory]{Name: "in-memory", New: sagatest.NewInMemory},
		sagatest.ContractChecks{
			{
				Method: "Compensate",
				Name:   "undoes-an-applied-step",
				Claim:  "Compensate undoes the step that was applied",
				Run: func(tb testing.TB, s saga.Contract, fx sagatest.ContractFixture) {
					tb.Helper()
					testkit.NoError(tb, s.Step(tb.Context(), fx.Value()), "a step applies")
					testkit.NoError(tb, s.Compensate(tb.Context(), fx.Value()),
						"the applied step is compensated")
					testkit.ErrorIs(tb, s.Compensate(tb.Context(), fx.Value()), sagatest.ErrNotApplied,
						"and compensating it again has nothing to undo")
				},
			},
			{
				Method: "State",
				Name:   "fingerprints-in-application-order",
				Claim:  "State fingerprints in application order",
				Run: func(tb testing.TB, s saga.Contract, fx sagatest.ContractFixture) {
					tb.Helper()
					before, err := s.State(tb.Context())
					testkit.NoError(tb, err, "the state is readable")

					first, second := fx.Value(), fx.ValueOther()
					testkit.NoError(tb, s.Step(tb.Context(), first), "the first step applies")
					testkit.NoError(tb, s.Step(tb.Context(), second), "and the second after it")

					stepped, err := s.State(tb.Context())
					testkit.NoError(tb, err, "the state is still readable")
					testkit.NotEqual(tb, stepped, before, "two applied steps changed the fingerprint")

					testkit.NoError(tb, s.Compensate(tb.Context(), second),
						"the newest step compensates")
					testkit.NoError(tb, s.Compensate(tb.Context(), first),
						"then the one before it")

					after, err := s.State(tb.Context())
					testkit.NoError(tb, err, "and the state is readable at the end")
					testkit.Equal(tb, after, before, "full compensation restored the fingerprint")
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

	sagatest.RunContract(t,
		sagatest.ContractHarness[*sagatest.InMemory]{Name: "in-memory", New: sagatest.NewInMemory},
		sagatest.ContractSuite.Without(sagatest.ContractSuite.Checks.Step.Smoke()),
	)
}
