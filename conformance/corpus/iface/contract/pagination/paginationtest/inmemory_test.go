// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package paginationtest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/pagination"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/pagination/paginationtest"
)

// pagination is the model tier's under ADR-0018: `AUTO-PAGINATOR-NO-DUPLICATES`
// and `AUTO-PAGINATOR-RESUMABLE` state it.
//
// The fixture's reader role takes a key and no page token — `cursor=Cursor` is
// an opaque param naming something the signature does not carry — so nothing
// here can walk pages at all. The paging claims are stated against
// composite/paginated-reader, whose reader takes the cursor the contract talks
// about.
func TestContractContract(t *testing.T) {
	t.Parallel()

	fixture := paginationtest.DefaultContractFixture()

	paginationtest.AssertContractContract(t,
		paginationtest.ContractSubject("in-memory", func() pagination.Contract {
			return paginationtest.NewInMemory()
		}),
		paginationtest.ContractSeed(func(_ context.Context, subject pagination.Contract) error {
			// The reader role is the only one declared, so nothing is derived
			// to seed through and the hit path is unreachable without this.
			subject.(*paginationtest.InMemory).Store(
				pagination.Value{Key: fixture.Key, Body: "seeded"},
			)
			return nil
		}),
		paginationtest.ContractOnGet("returns what was seeded", func(
			tb testing.TB, subject pagination.Contract, key string,
		) {
			tb.Helper()
			got, err := subject.Get(tb.Context(), key)
			testkit.NoError(tb, err, "a seeded key is found")
			testkit.Equal(tb, got.Body, "seeded", "and carries what was written")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	paginationtest.AssertContractContract(t,
		paginationtest.ContractSubject("in-memory", func() pagination.Contract {
			return paginationtest.NewInMemory()
		}),
		paginationtest.ContractWithout("Get/smoke"),
		paginationtest.ContractWithoutDouble(),
	)
}
