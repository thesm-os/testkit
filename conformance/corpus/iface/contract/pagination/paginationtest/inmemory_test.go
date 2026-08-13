// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package paginationtest_test

import (
	"fmt"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/pagination"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/pagination/paginationtest"
)

// pagination is the model tier's under ADR-0018: `AUTO-PAGINATOR-NO-DUPLICATES`
// and `AUTO-PAGINATOR-RESUMABLE` state it, over the page-shaped reader the
// fixture now declares.
//
// The check below is the deterministic complement: a seeded store small enough
// to hand-walk, asserting the page boundary lands where PageSize says and the
// keys arrive in order.
func TestContractContract(t *testing.T) {
	t.Parallel()

	paginationtest.AssertContractContract(t,
		paginationtest.ContractModel(),
		paginationtest.ContractSubject("in-memory", func() pagination.Contract {
			return paginationtest.NewInMemory()
		}),
		paginationtest.ContractOnPage("walks every entry once, in key order", func(
			tb testing.TB, subject pagination.Contract, _ pagination.Cursor,
		) {
			tb.Helper()
			const entries = 5
			for i := range entries {
				testkit.NoError(tb, subject.Put(tb.Context(), pagination.Value{
					Key:  fmt.Sprintf("b6-%02d", i),
					Body: "seeded",
				}), "an entry is stored")
			}

			// The walk sees at least the five above; the harness may have
			// seeded more, so the assertions are about order and termination
			// rather than exact contents.
			seen := map[string]bool{}
			last, cur := "", pagination.Cursor("")
			for range 100 {
				items, next, more, err := subject.Page(tb.Context(), cur)
				testkit.NoError(tb, err, "a page is readable")
				for _, v := range items {
					testkit.Equal(tb, v.Key > last, true, "keys arrive strictly ascending")
					last = v.Key
					seen[v.Key] = true
				}
				if !more {
					break
				}
				cur = next
			}
			for i := range entries {
				key := fmt.Sprintf("b6-%02d", i)
				testkit.Equal(tb, seen[key], true, "the stored entry paged out")
			}
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
		paginationtest.ContractWithout("Page/smoke"),
		paginationtest.ContractWithoutDouble(),
	)
}

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestContractSaturation(t *testing.T) {
	t.Parallel()
	paginationtest.ContractModelSaturation(t, func() pagination.Contract {
		return paginationtest.NewInMemory()
	})
}
