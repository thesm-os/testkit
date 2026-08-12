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
func TestContractContract(t *testing.T) {
	t.Parallel()

	codectest.AssertContractContract(t,
		codectest.ContractModel(),
		codectest.ContractSubject("in-memory", func() codec.Contract {
			return codectest.NewInMemory()
		}),
		// Dropped rather than satisfied: base64 encodes every string, so
		// there is no input Encode refuses. Decode keeps its check — a
		// string the forward transform could not have produced is a real
		// miss, and that is the asymmetry the pair has.
		codectest.ContractWithout("Encode/an error carries the zero value"),
		codectest.ContractOnDecode("undoes exactly what Encode did", func(
			tb testing.TB, subject codec.Contract, in string,
		) {
			tb.Helper()
			// The pair, stated once. `fidelity=exact` is the claim that this
			// round trip is the identity — a lossy codec would declare the
			// weaker form and this assertion would be wrong for it.
			encoded, err := subject.Encode(tb.Context(), in)
			testkit.NoError(tb, err, "the forward transform succeeds")

			decoded, err := subject.Decode(tb.Context(), encoded)
			testkit.NoError(tb, err, "and the inverse undoes it")
			testkit.Equal(tb, decoded, in, "exact fidelity is the identity")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	codectest.AssertContractContract(t,
		codectest.ContractSubject("in-memory", func() codec.Contract {
			return codectest.NewInMemory()
		}),
		// Dropped rather than satisfied: base64 encodes every string, so
		// there is no input Encode refuses. Decode keeps its check — a
		// string the forward transform could not have produced is a real
		// miss, and that is the asymmetry the pair has.
		codectest.ContractWithout("Encode/an error carries the zero value"),
		codectest.ContractWithoutDouble(),
	)
}
