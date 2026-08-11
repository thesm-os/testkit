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

	poisonaccessortest.AssertPoisonAccessorContract(t,
		poisonaccessortest.PoisonAccessorSubject("in-memory", func() poisonaccessor.PoisonAccessor {
			return poisonaccessortest.NewInMemory()
		}),
		poisonaccessortest.PoisonAccessorOnErr("answers nil while healthy", func(
			tb testing.TB, subject poisonaccessor.PoisonAccessor,
		) {
			tb.Helper()
			testkit.NoError(tb, subject.Err(), "a fresh subject reports no failure")
		}),
	)
}

// The latch: once poisoned, every later call reports the same error. Reading it
// must not clear it, or the accessor is a queue and a second reader sees health
// that is not there.
func TestErrLatchesOncePoisoned(t *testing.T) {
	t.Parallel()

	s := poisonaccessortest.NewInMemory()
	testkit.NoError(t, s.Err(), "a fresh subject is healthy")

	s.Poison()
	testkit.ErrorIs(t, s.Err(), poisonaccessor.ErrPoisoned, "the failure is reported")
	testkit.ErrorIs(t, s.Err(), poisonaccessor.ErrPoisoned,
		"and again, because reading a latch does not clear it")
	testkit.Equal(t, s.Calls(), 3, "each read reached the subject rather than a cached answer")
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
