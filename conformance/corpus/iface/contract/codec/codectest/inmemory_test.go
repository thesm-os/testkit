// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package codectest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/codec"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/codec/codectest"
)

// The generated contract, run against the in-memory subject.
//
// Nothing in either signature pairs Encode with Decode, so the round trip is
// the row's to state: the generator derives each method's own family and has
// no way to know one undoes the other.
func TestContractContract(t *testing.T) {
	t.Parallel()

	codectest.RunContract(t,
		codectest.ContractHarness[*codectest.InMemory]{Name: "in-memory", New: codectest.NewInMemory},
		codectest.ContractChecks{
			{
				Method: "Decode",
				Name:   "undoes-encode",
				Claim:  "Decode undoes exactly what Encode did",
				Run: func(tb testing.TB, s codec.Contract, fx codectest.ContractFixture) {
					tb.Helper()
					// The pair, stated once. `fidelity=exact` is the claim that
					// this round trip is the identity — a lossy codec would
					// declare the weaker form and this assertion would be wrong
					// for it.
					encoded, err := s.Encode(tb.Context(), fx.In())
					testkit.NoError(tb, err, "the forward transform succeeds")

					decoded, err := s.Decode(tb.Context(), encoded)
					testkit.NoError(tb, err, "and the inverse undoes it")
					testkit.Equal(tb, decoded, fx.In(), "exact fidelity is the identity")
				},
			},
		},
	)
}
