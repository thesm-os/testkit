// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package castest_test

import (
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/cas"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/cas/castest"
)

// cas is the model tier's under ADR-0018: `AUTO-CAS-ATOMIC-ONE-WINNER` states
// it, and stating it needs accumulated version state — a stale revision to
// present, which only a sequence of writes produces.
//
// `version=Version` is an opaque param naming a field rather than a callable,
// so there is no partner to call and no verdict to read. The suite tier gets
// the signature-derived family and says so.
func TestContractContract(t *testing.T) {
	t.Parallel()

	castest.AssertContractContract(t,
		castest.ContractSubject("in-memory", func() cas.Contract {
			return castest.NewInMemory()
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	castest.AssertContractContract(t,
		castest.ContractSubject("in-memory", func() cas.Contract {
			return castest.NewInMemory()
		}),
		castest.ContractWithout("Put/smoke"),
		castest.ContractWithoutDouble(),
	)
}
