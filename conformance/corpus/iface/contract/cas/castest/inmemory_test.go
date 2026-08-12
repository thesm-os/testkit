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

	// The derived fixture cannot know the cell's dialect: a fresh cell
	// accepts version zero and nothing else, so the seeded values say so —
	// the second value stays at zero too, because every check gets a fresh
	// subject whose cell is back at the start.
	fixture := castest.DefaultContractFixture()
	fixture.V.Version = 0
	fixture.VOther.Version = 0

	castest.AssertContractContract(t,
		castest.ContractModel(),
		castest.ContractSubject("in-memory", func() cas.Contract {
			return castest.NewInMemory()
		}),
		castest.ContractWithFixture(fixture),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	fixture := castest.DefaultContractFixture()
	fixture.V.Version = 0
	fixture.VOther.Version = 0

	castest.AssertContractContract(t,
		castest.ContractSubject("in-memory", func() cas.Contract {
			return castest.NewInMemory()
		}),
		castest.ContractWithFixture(fixture),
		castest.ContractWithout("Put/smoke"),
		castest.ContractWithoutDouble(),
	)
}
