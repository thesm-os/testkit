// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package validatestest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/validates"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/validates/validatestest"
)

// The whole wiring a consumer writes: one subject, and the checks the generator
// has no classification to derive.
//
// Every value the run uses is derived — from each parameter's own type — and
// so is the double, which comes from the //testkit:stub on the same interface.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	validatestest.RunMixed(t,
		validatestest.MixedHarness[*validatestest.InMemory]{Name: "in-memory", New: validatestest.NewInMemory},
		validatestest.MixedChecks{
			{
				Method: "Store",
				Name:   "refuses-what-validate-refuses",
				Claim:  "Store refuses what its own validator refuses",
				Run: func(tb testing.TB, s validates.Mixed, fx validatestest.MixedFixture) {
					tb.Helper()
					// The mixin's own law, written by hand until the generator
					// reads the classification: a payload with no key is one
					// Validate rejects, and Store must not take it.
					invalid := validates.Payload{Body: fx.Payload().Body}
					testkit.ErrorIs(tb, s.Validate(invalid), validatestest.ErrInvalid,
						"a payload with no key does not validate")
					testkit.ErrorIs(tb, s.Store(tb.Context(), invalid), validatestest.ErrInvalid,
						"and Store refuses it for the same reason")
				},
			},
			{
				Method: "Read",
				Name:   "reads-back-what-store-wrote",
				Claim:  "Read returns what Store wrote",
				Run: func(tb testing.TB, s validates.Mixed, fx validatestest.MixedFixture) {
					tb.Helper()
					// Writes its own precondition rather than assuming one: a
					// check that read what something else left behind would
					// break the moment a subject supplied its own.
					want := validates.Payload{Key: fx.Key(), Body: "read-after-write"}
					testkit.NoError(tb, s.Store(tb.Context(), want),
						"a valid payload stores under its own key")

					got, err := s.Read(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "and is found under it")
					testkit.Equal(tb, got, want, "and comes back whole")
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

	validatestest.RunMixed(t,
		validatestest.MixedHarness[*validatestest.InMemory]{Name: "in-memory", New: validatestest.NewInMemory},
		validatestest.MixedSuite.Without(validatestest.MixedSuite.Checks.Store.Smoke()),
	)
}
