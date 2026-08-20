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
// The check below is the deterministic complement: a store it seeded itself,
// small enough to hand-walk, asserting the keys arrive in order and the walk
// terminates.
func TestContractContract(t *testing.T) {
	t.Parallel()

	paginationtest.RunContract(t,
		paginationtest.ContractHarness[*paginationtest.InMemory]{Name: "in-memory", New: paginationtest.NewInMemory},
		paginationtest.ContractChecks{
			{
				Method: "Page",
				Name:   "walks-in-key-order",
				Claim:  "Page walks every entry once, in key order",
				Run: func(tb testing.TB, s pagination.Contract, fx paginationtest.ContractFixture) {
					tb.Helper()
					const entries = 5
					for i := range entries {
						testkit.NoError(tb, s.Put(tb.Context(), pagination.Value{
							Key:  fmt.Sprintf("b6-%02d", i),
							Body: "seeded",
						}), "an entry is stored")
					}

					// The walk sees exactly what this row wrote: nothing seeds a
					// subject now but its own constructor, and this one starts
					// empty.
					seen := map[string]bool{}
					last, cur := "", pagination.Cursor("")
					for range 100 {
						items, next, more, err := s.Page(tb.Context(), cur)
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
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	paginationtest.RunContract(t,
		paginationtest.ContractHarness[*paginationtest.InMemory]{Name: "in-memory", New: paginationtest.NewInMemory},
		paginationtest.ContractSuite.Without(paginationtest.ContractSuite.Checks.Page.Smoke()),
	)
}
