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
// nothing ever trips and report success for every implementation. What a check
// written here can do is name the downstream, which is the fixture's own way of
// saying "this one is unwell" — the extension point reaching what derivation
// could not.
func TestContractContract(t *testing.T) {
	t.Parallel()

	circuitbreakertest.AssertContractContract(t,
		circuitbreakertest.ContractModel(),
		circuitbreakertest.ContractSubject("in-memory", func() circuitbreaker.Contract {
			return circuitbreakertest.NewInMemory(threshold)
		}),
		circuitbreakertest.ContractOnRun("stops calling a downstream that keeps failing", func(
			tb testing.TB, subject circuitbreaker.Contract, key string,
		) {
			tb.Helper()
			// The lever is the key, so the claim is statable through the
			// interface after all: a breaker guards something the caller names,
			// and asking for the unwell one is how a run induces the failure.
			for range threshold {
				testkit.ErrorIs(tb, subject.Run(tb.Context(), circuitbreakertest.UnwellKey),
					circuitbreakertest.ErrDownstream,
					"a call under the threshold reaches the downstream")
			}
			testkit.ErrorIs(tb, subject.Run(tb.Context(), circuitbreakertest.UnwellKey),
				circuitbreakertest.ErrOpen,
				"and the call after it is refused by the breaker")

			testkit.NoError(tb, subject.Run(tb.Context(), key),
				"while a healthy downstream is still reachable")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	circuitbreakertest.AssertContractContract(t,
		circuitbreakertest.ContractSubject("in-memory", func() circuitbreaker.Contract {
			return circuitbreakertest.NewInMemory(3)
		}),
		circuitbreakertest.ContractWithout("Run/smoke"),
		circuitbreakertest.ContractWithoutDouble(),
	)
}
