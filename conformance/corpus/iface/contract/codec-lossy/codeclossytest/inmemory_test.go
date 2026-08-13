// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package codeclossytest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	codeclossy "go.thesmos.sh/testkit/conformance/corpus/iface/contract/codec-lossy"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/codec-lossy/codeclossytest"
)

// The generated contract, run against the in-memory subject.
func TestContractContract(t *testing.T) {
	t.Parallel()

	codeclossytest.AssertContractContract(t,
		codeclossytest.ContractModel(),
		codeclossytest.ContractSubject("in-memory", func() codeclossy.Contract {
			return codeclossytest.NewInMemory()
		}),
		codeclossytest.ContractOnDecode("agrees with itself on the second pass", func(
			tb testing.TB, subject codeclossy.Contract, in string,
		) {
			tb.Helper()
			// `fidelity=lossy` is the weaker claim, stated once: not the
			// identity, but stability — whatever the fold lost stays lost,
			// and re-encoding the recovery reproduces the first encoding.
			encoded, err := subject.Encode(tb.Context(), in)
			testkit.NoError(tb, err, "the forward transform succeeds")

			decoded, err := subject.Decode(tb.Context(), encoded)
			testkit.NoError(tb, err, "the inverse recovers what survived")

			again, err := subject.Encode(tb.Context(), decoded)
			testkit.NoError(tb, err, "a second pass still encodes")
			testkit.Equal(tb, again, encoded, "and loses nothing new")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	codeclossytest.AssertContractContract(t,
		codeclossytest.ContractSubject("in-memory", func() codeclossy.Contract {
			return codeclossytest.NewInMemory()
		}),
		codeclossytest.ContractWithoutDouble(),
	)
}
