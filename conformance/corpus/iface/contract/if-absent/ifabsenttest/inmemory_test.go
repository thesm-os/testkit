// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ifabsenttest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	ifabsent "go.thesmos.sh/testkit/conformance/corpus/iface/contract/if-absent"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/if-absent/ifabsenttest"
)

// if-absent is the suite tier's under ADR-0018: no [engine/model/law] property
// states "a second write for one key is refused", and the claim needs nothing
// the tier cannot produce — one subject, two calls, the same value both times.
//
// The generated check writes VOther rather than V. The harness seeds every
// fresh subject through Put itself, so handed V the first write would already
// be the second and a correct store would fail the line meant to succeed.
func TestContractContract(t *testing.T) {
	t.Parallel()

	ifabsenttest.AssertContractContract(t,
		ifabsenttest.ContractSubject("in-memory", func() ifabsent.Contract {
			return ifabsenttest.NewInMemory()
		}),
		ifabsenttest.ContractOnPut("refuses the key rather than the call", func(
			tb testing.TB, subject ifabsent.Contract, v ifabsent.Value,
		) {
			tb.Helper()
			// A store refusing every write after the first passes the generated
			// check without holding a key at all, which is the reading of
			// "refused" the contract does not mean.
			testkit.Error(tb, subject.Put(tb.Context(), v),
				"the seeded key is refused")
			testkit.NoError(tb, subject.Put(tb.Context(), ifabsent.Value{Key: v.Key + "-fresh", Body: v.Body}),
				"and a key nothing holds is still accepted")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	ifabsenttest.AssertContractContract(t,
		ifabsenttest.ContractSubject("in-memory", func() ifabsent.Contract {
			return ifabsenttest.NewInMemory()
		}),
		ifabsenttest.ContractWithout("Put/smoke"),
		ifabsenttest.ContractWithoutDouble(),
	)
}
