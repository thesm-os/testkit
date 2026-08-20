// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package poisonaccessortest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/poisonaccessor"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/poisonaccessor/poisonaccessortest"
)

// latches is the shape's whole law, written once and run against both births.
//
// A latch is a claim about two calls — the second reports what the first did —
// and every generated check makes one call against a fresh subject. Nothing
// here is a gap in the harness: a single call cannot observe that a state
// persists, so the law belongs to this package and the signature owes only the
// smoke call.
var latches = poisonaccessortest.PoisonAccessorChecks{
	{
		Method: "Err",
		Name:   "latches-what-it-reports",
		Claim:  "Err latches whatever it reports",
		Run: func(tb testing.TB, s poisonaccessor.PoisonAccessor, fx poisonaccessortest.PoisonAccessorFixture) {
			tb.Helper()
			// Stated across two births because Err is the only method:
			// nothing a caller does moves a healthy subject into failure, so
			// the failed state is a constructor's to build. What both share is
			// that the answer does not change on being read.
			first := s.Err()
			testkit.Equal(tb, s.Err(), first,
				"a second read reports what the first did")
		},
	},
}

func TestPoisonAccessorContract(t *testing.T) {
	t.Parallel()

	poisonaccessortest.RunPoisonAccessor(
		t,
		poisonaccessortest.PoisonAccessorHarness[*poisonaccessortest.InMemory]{
			Name: "in-memory",
			New:  poisonaccessortest.NewInMemory,
		},
		latches,
	)
	// The born-failed arm: AUTO-POISON-NIL-ON-FRESH is the claim that a fresh
	// instance reports nothing, and this constructor exists to violate exactly
	// that — it exercises the latch, not the birth.
	poisonaccessortest.RunPoisonAccessor(t,
		poisonaccessortest.PoisonAccessorHarness[*poisonaccessortest.InMemory]{
			Name: "in-memory, already failed", New: poisonaccessortest.Poisoned,
		},
		latches,
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestPoisonAccessorContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	poisonaccessortest.RunPoisonAccessor(
		t,
		poisonaccessortest.PoisonAccessorHarness[*poisonaccessortest.InMemory]{
			Name: "in-memory",
			New:  poisonaccessortest.NewInMemory,
		},
		poisonaccessortest.PoisonAccessorSuite.Without(poisonaccessortest.PoisonAccessorSuite.Checks.Err.Smoke()),
	)
}

// No saturation prover: this fixture binds no laws, and that is the point of
// the axis. `poisonaccessor` is a signature — a nullary bare-error callable —
// and the latch is a claim made with `poisonable induce=`, which the mixin
// fixture next door makes. A detector fixture proves the stamp lands; the
// laws that stamp once earned failed every correct close-once teardown.
