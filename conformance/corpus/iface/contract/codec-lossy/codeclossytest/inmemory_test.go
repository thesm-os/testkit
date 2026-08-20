// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package codeclossytest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	codeclossy "go.thesmos.sh/testkit/conformance/corpus/iface/contract/codec-lossy"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/codec-lossy/codeclossytest"
)

// The same two signatures as codec, and a different claim about them — which
// is the point of the pair: `fidelity` is not in either signature, so the
// generated families are identical and only the row below tells them apart.
func TestContractContract(t *testing.T) {
	t.Parallel()

	codeclossytest.RunContract(t,
		codeclossytest.ContractHarness[*codeclossytest.InMemory]{Name: "in-memory", New: codeclossytest.NewInMemory},
		codeclossytest.ContractChecks{
			{
				Method: "Decode",
				Name:   "second-pass-agrees",
				Claim:  "Decode agrees with itself on the second pass",
				Run: func(tb testing.TB, s codeclossy.Contract, fx codeclossytest.ContractFixture) {
					tb.Helper()
					// `fidelity=lossy` is the weaker claim, stated once: not the
					// identity, but stability — whatever the fold lost stays
					// lost, and re-encoding the recovery reproduces the first
					// encoding.
					encoded, err := s.Encode(tb.Context(), fx.In())
					testkit.NoError(tb, err, "the forward transform succeeds")

					decoded, err := s.Decode(tb.Context(), encoded)
					testkit.NoError(tb, err, "the inverse recovers what survived")

					again, err := s.Encode(tb.Context(), decoded)
					testkit.NoError(tb, err, "a second pass still encodes")
					testkit.Equal(tb, again, encoded, "and loses nothing new")
				},
			},
		},
	)
}
