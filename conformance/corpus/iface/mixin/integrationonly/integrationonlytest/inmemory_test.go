// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package integrationonlytest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/integrationonly"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/integrationonly/integrationonlytest"
)

// integrationonly generates no check, and it is the one classification whose
// absence is not a judgement about what can be asserted.
//
// The fixture's own doc says the mixin "gates the generated subtest behind a
// build tag" — output routing, not a template: it needs a third output whose
// suffix carries the constraint and Layout to route it there. Until that
// exists there is nothing for a check template to say.
func TestMixedContract(t *testing.T) {
	// Not parallel, because t.Setenv forbids it — and the variable is the
	// point: every check on Connect sits behind the integration guard, so a run
	// that does not set it skips the whole method and exercises nothing.
	//
	// Which is what a consumer's integration stage does, and what their unit
	// stage deliberately does not.
	t.Setenv("TESTKIT_INTEGRATION", "1")

	integrationonlytest.RunMixed(
		t,
		integrationonlytest.MixedHarness[*integrationonlytest.InMemory]{
			Name: "in-memory",
			New:  integrationonlytest.NewInMemory,
		},
		integrationonlytest.MixedChecks{
			{
				Method: "Connect",
				Name:   "refuses-an-unparseable-target",
				Claim:  "Connect refuses a target it cannot parse",
				Run: func(tb testing.TB, s integrationonly.Mixed, fx integrationonlytest.MixedFixture) {
					tb.Helper()
					// A well-formed target is the row's to supply: the derived
					// draw is a plausible string rather than a URL, which is
					// the half of this claim derivation cannot reach.
					testkit.NoError(tb, s.Connect(tb.Context(), "postgres://localhost/primary"),
						"a target with a scheme is accepted")

					// A subject that accepted any string would fail at first
					// use rather than at configuration, which is the wrong end
					// of the deployment.
					testkit.ErrorIs(tb, s.Connect(tb.Context(), "not-a-dsn"), integrationonlytest.ErrBadDSN,
						"a target with no scheme is refused")
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

	integrationonlytest.RunMixed(
		t,
		integrationonlytest.MixedHarness[*integrationonlytest.InMemory]{
			Name: "in-memory",
			New:  integrationonlytest.NewInMemory,
		},
		integrationonlytest.MixedSuite.Without(integrationonlytest.MixedSuite.Checks.Connect.Smoke()),
	)
}
