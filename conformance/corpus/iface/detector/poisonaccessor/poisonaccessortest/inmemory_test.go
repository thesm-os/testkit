// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package poisonaccessortest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/poisonaccessor"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/poisonaccessor/poisonaccessortest"
)

// One generated check, and the shape's whole point is out of its reach.
//
// A latch is a claim about two calls — the second reports what the first did —
// and every generated check makes one call against a fresh subject. Nothing here
// is a gap in the harness: a single call cannot observe that a state persists,
// so the law belongs to this package and the signature owes only the smoke call.
func TestPoisonAccessorContract(t *testing.T) {
	t.Parallel()

	onErr := poisonaccessortest.PoisonAccessorOnErr("latches whatever it reports", func(
		tb testing.TB, subject poisonaccessor.PoisonAccessor,
	) {
		tb.Helper()
		// The latch, stated across two subjects because Err is the only
		// method: nothing a caller does moves a healthy one into failure,
		// so the failed state is a factory's to build. What both share is
		// that the answer does not change on being read.
		first := subject.Err()
		testkit.Equal(tb, subject.Err(), first,
			"a second read reports what the first did")
	})

	poisonaccessortest.AssertPoisonAccessorContract(t,
		poisonaccessortest.PoisonAccessorModel(),
		poisonaccessortest.PoisonAccessorSubject("in-memory", func() poisonaccessor.PoisonAccessor {
			return poisonaccessortest.NewInMemory()
		}),
		onErr,
	)
	// The born-failed arm runs without the model tier: AUTO-POISON-NIL-ON-FRESH
	// is the claim that a fresh instance reports nothing, and this factory
	// exists to violate exactly that — it exercises the latch, not the birth.
	poisonaccessortest.AssertPoisonAccessorContract(t,
		poisonaccessortest.PoisonAccessorSubject("in-memory, already failed", func() poisonaccessor.PoisonAccessor {
			return poisonaccessortest.Poisoned()
		}),
		onErr,
	)
}

// Declining the double is separate from dropping a check.
func TestPoisonAccessorContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	poisonaccessortest.AssertPoisonAccessorContract(t,
		poisonaccessortest.PoisonAccessorSubject("in-memory", func() poisonaccessor.PoisonAccessor {
			return poisonaccessortest.NewInMemory()
		}),
		poisonaccessortest.PoisonAccessorWithout("Err/smoke"),
		poisonaccessortest.PoisonAccessorWithoutDouble(),
	)
}
