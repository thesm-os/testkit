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
// can be asked for, because a check runs against every subject a consumer
// declares and again through the double — where a test in this package runs
// against the one implementation it happens to hold.
//
// Run is classified writer, so the harness seeds through it: every check meets
// a workflow already in Draft, and the check's own call takes it to Live.
func TestContractContract(t *testing.T) {
	t.Parallel()

	workflowtest.AssertContractContract(t,
		workflowtest.ContractSubject("in-memory", func() workflow.Contract {
			return workflowtest.NewInMemory()
		}),
		workflowtest.ContractOnRun("refuses a transition out of the last state", func(
			tb testing.TB, subject workflow.Contract, key string,
		) {
			tb.Helper()
			// The seed took the key to Draft, so this reaches Live — the last
			// state `transitions=Draft>Live` declares.
			testkit.NoError(tb, subject.Run(tb.Context(), key), "the declared transition runs")
			testkit.ErrorIs(tb, subject.Run(tb.Context(), key), workflowtest.ErrTerminal,
				"and there is no transition out of where it left the key")
		}),
		workflowtest.ContractOnRun("advances one key without advancing another", func(
			tb testing.TB, subject workflow.Contract, key string,
		) {
			tb.Helper()
			// A subject holding one state for the whole workflow rather than
			// one per key passes every single-key check and settles every
			// caller's work at once.
			other := key + "-other"
			testkit.NoError(tb, subject.Run(tb.Context(), other), "a fresh key starts")
			testkit.NoError(tb, subject.Run(tb.Context(), other), "and advances to the last state")
			testkit.NoError(tb, subject.Run(tb.Context(), key),
				"while the seeded key still has its own transition left")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	workflowtest.AssertContractContract(t,
		workflowtest.ContractSubject("in-memory", func() workflow.Contract {
			return workflowtest.NewInMemory()
		}),
		workflowtest.ContractWithout("Run/smoke"),
		workflowtest.ContractWithoutDouble(),
	)
}
