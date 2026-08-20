// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package workflowtest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/workflow"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/workflow/workflowtest"
)

// workflow is the model tier's under ADR-0018: `AUTO-VALID-TRANSITION` states
// it.
//
// What is stated here rather than in a package test is everything the interface
// can be asked for, because a row runs against every subject a consumer
// declares and again through the double — where a test in this package runs
// against the one implementation it happens to hold.
func TestContractContract(t *testing.T) {
	t.Parallel()

	workflowtest.RunContract(t,
		workflowtest.ContractHarness[*workflowtest.InMemory]{Name: "in-memory", New: workflowtest.NewInMemory},
		// A key nothing has run is in the first state, not absent: this
		// workflow has no "unknown", so State answers Draft rather than the
		// zero. The reader shape cannot see that, and the subject is not
		// wrong — which is what a drop is for.
		workflowtest.ContractSuite.Without(workflowtest.ContractSuite.Checks.State.Miss()),
		workflowtest.ContractChecks{
			{
				Method: "Run",
				Name:   "refuses-a-transition-out-of-the-last-state",
				Claim:  "Run refuses a transition out of the last state",
				Run: func(tb testing.TB, s workflow.Contract, fx workflowtest.ContractFixture) {
					tb.Helper()
					// `transitions=Draft>Live` declares two states, so the row
					// walks the key to the last one and asks for one more.
					testkit.NoError(tb, s.Run(tb.Context(), fx.Key()), "a fresh key starts")
					testkit.NoError(tb, s.Run(tb.Context(), fx.Key()), "the declared transition runs")
					testkit.ErrorIs(tb, s.Run(tb.Context(), fx.Key()), workflowtest.ErrTerminal,
						"and there is no transition out of where it left the key")
				},
			},
			{
				Method: "Run",
				Name:   "advances-one-key-not-another",
				Claim:  "Run advances one key without advancing another",
				Run: func(tb testing.TB, s workflow.Contract, fx workflowtest.ContractFixture) {
					tb.Helper()
					// A subject holding one state for the whole workflow rather
					// than one per key passes every single-key check and settles
					// every caller's work at once.
					testkit.NoError(tb, s.Run(tb.Context(), fx.KeyOther()), "a fresh key starts")
					testkit.NoError(tb, s.Run(tb.Context(), fx.KeyOther()),
						"and advances to the last state")
					testkit.NoError(tb, s.Run(tb.Context(), fx.Key()),
						"while another key still has both its transitions left")
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

	workflowtest.RunContract(t,
		workflowtest.ContractHarness[*workflowtest.InMemory]{Name: "in-memory", New: workflowtest.NewInMemory},
		workflowtest.ContractSuite.Without(
			workflowtest.ContractSuite.Checks.Run.Smoke(),
			// The same drop the run above makes, for the same reason.
			workflowtest.ContractSuite.Checks.State.Miss(),
		),
	)
}
