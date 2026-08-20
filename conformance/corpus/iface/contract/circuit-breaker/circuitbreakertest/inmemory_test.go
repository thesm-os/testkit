// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package circuitbreakertest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	circuitbreaker "go.thesmos.sh/testkit/conformance/corpus/iface/contract/circuit-breaker"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/circuit-breaker/circuitbreakertest"
)

// threshold is how many consecutive failures trip this subject's breaker.
const threshold = 3

// circuit-breaker is owned by no tier under ADR-0018, and the gate reports that
// as a law to write rather than a check to invent.
//
// What a *generated* check cannot do is induce the failure: nothing in the
// directive says which input fails, so a derived one would drive a breaker
// nothing ever trips and report success for every implementation. What a row
// can do is name the downstream, which is the fixture's own way of saying
// "this one is unwell".
func TestContractContract(t *testing.T) {
	t.Parallel()

	circuitbreakertest.RunContract(t,
		circuitbreakertest.ContractHarness[*circuitbreakertest.InMemory]{
			Name: "in-memory",
			New:  func() *circuitbreakertest.InMemory { return circuitbreakertest.NewInMemory(threshold) },
		},
		circuitbreakertest.ContractChecks{
			{
				Method: "Run",
				Name:   "stops-calling-a-failing-downstream",
				Claim:  "Run stops calling a downstream that keeps failing",
				Run: func(tb testing.TB, s circuitbreaker.Contract, fx circuitbreakertest.ContractFixture) {
					tb.Helper()
					// The lever is the key, so the claim is statable through the
					// interface after all: a breaker guards something the caller
					// names, and asking for the unwell one is how a run induces
					// the failure.
					for range threshold {
						testkit.ErrorIs(tb, s.Run(tb.Context(), circuitbreakertest.UnwellKey),
							circuitbreakertest.ErrDownstream,
							"a call under the threshold reaches the downstream")
					}
					testkit.ErrorIs(tb, s.Run(tb.Context(), circuitbreakertest.UnwellKey),
						circuitbreakertest.ErrOpen,
						"and the call after it is refused by the breaker")

					testkit.NoError(tb, s.Run(tb.Context(), fx.Key()),
						"while a healthy downstream is still reachable")
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

	circuitbreakertest.RunContract(t,
		circuitbreakertest.ContractHarness[*circuitbreakertest.InMemory]{
			Name: "in-memory",
			New:  func() *circuitbreakertest.InMemory { return circuitbreakertest.NewInMemory(threshold) },
		},
		circuitbreakertest.ContractSuite.Without(circuitbreakertest.ContractSuite.Checks.Run.Smoke()),
	)
}
