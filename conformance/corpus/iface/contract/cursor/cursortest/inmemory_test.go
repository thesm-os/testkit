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
// Neither role writes, so nothing is derived to seed through — and unlike a
// store, nothing needs to be: a cursor over an empty sequence still has to
// report exhaustion rather than panic, which is what the signature-derived
// family drives. The subject is built with values so the checks read something.
func TestContractContract(t *testing.T) {
	t.Parallel()

	cursortest.AssertContractContract(t,
		cursortest.ContractModel(),
		cursortest.ContractSubject("in-memory", func() cursor.Contract {
			return cursortest.NewInMemory(
				cursor.Value{Key: "a", Body: "one"},
				cursor.Value{Key: "b", Body: "two"},
			)
		}),
		cursortest.ContractSubject("in-memory, empty", func() cursor.Contract {
			return cursortest.NewInMemory()
		}),
		cursortest.ContractOnNext("refuses a read after the cursor is closed", func(
			tb testing.TB, subject cursor.Contract,
		) {
			tb.Helper()
			// Exhausted is "you have everything"; closed is "you gave up the
			// right to ask". A cursor reporting the first for the second hides
			// a bug in the caller's own control flow.
			testkit.NoError(tb, subject.Close(tb.Context()), "the cursor closes")

			_, ok, err := subject.Next(tb.Context())
			testkit.ErrorIs(tb, err, cursor.ErrClosed, "and a read after it is refused")
			testkit.False(tb, ok, "with no value beside the error")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	cursortest.AssertContractContract(t,
		cursortest.ContractSubject("in-memory", func() cursor.Contract {
			return cursortest.NewInMemory()
		}),
		cursortest.ContractWithout("Next/smoke"),
		cursortest.ContractWithoutDouble(),
	)
}
