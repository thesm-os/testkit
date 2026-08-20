// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cursortest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/cursor"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/cursor/cursortest"
)

// cursor is the model tier's under ADR-0018: `AUTO-CURSOR-NEXT-AFTER-CLOSE` and
// `AUTO-CURSOR-CLOSE-IDEMPOTENT` state it.
//
// Neither role writes, and unlike a store nothing needs to: a cursor over an
// empty sequence still has to report exhaustion rather than panic, which is
// what the signature-derived family drives. Two subjects, so both the loaded
// and the empty case run.
func TestContractContract(t *testing.T) {
	t.Parallel()

	cursortest.RunContract(t,
		cursortest.ContractHarness[*cursortest.InMemory]{
			Name: "in-memory",
			New: func() *cursortest.InMemory {
				return cursortest.NewInMemory(
					cursor.Value{Key: "a", Body: "one"},
					cursor.Value{Key: "b", Body: "two"},
				)
			},
		},
		cursortest.ContractHarness[*cursortest.InMemory]{
			Name: "in-memory, empty",
			New:  func() *cursortest.InMemory { return cursortest.NewInMemory() },
		},
		cursortest.ContractChecks{
			{
				Method: "Next",
				Name:   "refuses-a-read-after-close",
				Claim:  "Next refuses a read after the cursor is closed",
				Run: func(tb testing.TB, s cursor.Contract, fx cursortest.ContractFixture) {
					tb.Helper()
					// Exhausted is "you have everything"; closed is "you gave up
					// the right to ask". A cursor reporting the first for the
					// second hides a bug in the caller's own control flow.
					testkit.NoError(tb, s.Close(tb.Context()), "the cursor closes")

					_, ok, err := s.Next(tb.Context())
					testkit.ErrorIs(tb, err, cursor.ErrClosed, "and a read after it is refused")
					testkit.False(tb, ok, "with no value beside the error")
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

	cursortest.RunContract(t,
		cursortest.ContractHarness[*cursortest.InMemory]{
			Name: "in-memory",
			New:  func() *cursortest.InMemory { return cursortest.NewInMemory() },
		},
		cursortest.ContractSuite.Without(cursortest.ContractSuite.Checks.Next.Smoke()),
	)
}
