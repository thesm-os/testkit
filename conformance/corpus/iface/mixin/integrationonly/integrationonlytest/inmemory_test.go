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
//
// Connect is classified writer, so the harness seeds every subject through it.
// A derived DSN is a plausible string rather than a URL, so the fixture supplies
// one this subject accepts — which is exactly what the generated seed-failure
// message asks for when it does not.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	integrationonlytest.AssertMixedContract(t,
		integrationonlytest.MixedSubject("in-memory", func() integrationonly.Mixed {
			return integrationonlytest.NewInMemory()
		}),
		integrationonlytest.MixedWithFixture(reachableFixture()),
	)
}

// The connection is observable, which the interface does not expose: Connect
// reports only whether it failed.
func TestConnectRecordsTheConnection(t *testing.T) {
	t.Parallel()

	s := integrationonlytest.NewInMemory()
	testkit.False(t, s.Connected(), "a fresh subject is disconnected")
	testkit.NoError(t, s.Connect(t.Context(), "postgres://localhost/db"), "connecting succeeds")
	testkit.True(t, s.Connected(), "and is observable")

	testkit.ErrorIs(t, integrationonlytest.NewInMemory().Connect(t.Context(), "not-a-dsn"),
		integrationonlytest.ErrBadDSN, "a malformed target is refused")
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	integrationonlytest.AssertMixedContract(t,
		integrationonlytest.MixedSubject("in-memory", func() integrationonly.Mixed {
			return integrationonlytest.NewInMemory()
		}),
		integrationonlytest.MixedWithFixture(reachableFixture()),
		integrationonlytest.MixedWithout("Connect/smoke"),
		integrationonlytest.MixedWithoutDouble(),
	)
}

// reachableFixture supplies targets this subject parses.
//
// Both halves: the alternate is a second well-formed DSN rather than a broken
// one, because it feeds checks that need a value Connect accepts and not one it
// refuses.
func reachableFixture() integrationonlytest.MixedFixture {
	f := integrationonlytest.DefaultMixedFixture()
	f.Dsn = "postgres://localhost/primary"
	f.DsnOther = "postgres://localhost/secondary"
	return f
}
