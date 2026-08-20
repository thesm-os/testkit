// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package errorstest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/errors"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/errors/errorstest"
)

// The errors mixin itself generates no check — the header records the gap —
// because the sentinels come from a different directive than the mixin.
//
// `//testkit:fault ErrNotFound ErrGone` names them, and that is the fault
// generator's key: it decides what the DOUBLE can be told to return. Which
// sentinel a real subject owes for which input is a separate declaration, and
// the miss half is now made through `//testkit:mixin notfound sentinel=` — so
// Get/miss is derived and only the second sentinel is left to state here.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	fx := errorstest.DefaultMixedFixture()

	errorstest.RunMixed(t,
		errorstest.MixedHarness[*errorstest.InMemory]{
			Name: "in-memory",
			// Nothing on this interface writes, so the seed is the
			// constructor's.
			New: func() *errorstest.InMemory {
				s := errorstest.NewInMemory()
				s.Put(fx.Key(), "stored")
				return s
			},
		},
		errorstest.MixedChecks{
			{
				Method: "Get",
				Name:   "removed-differs-from-missing",
				Claim:  "Get reports each declared sentinel for its own case",
				Run: func(tb testing.TB, s errors.Mixed, fx errorstest.MixedFixture) {
					tb.Helper()
					// Get/miss covers the absent key. This is the other
					// sentinel, and the reason the mixin is worth having: a
					// removed key is distinguishable from one that was never
					// there.
					_, err := s.Get(tb.Context(), errorstest.GoneKey())
					testkit.ErrorIs(tb, err, errors.ErrGone,
						"a removed key is distinguishable from a missing one")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	errorstest.RunMixed(t,
		errorstest.MixedHarness[*errorstest.InMemory]{Name: "in-memory", New: errorstest.NewInMemory},
		errorstest.MixedSuite.Without(errorstest.MixedSuite.Checks.Get.Smoke()),
	)
}
