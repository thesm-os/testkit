// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package puretest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/pure"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/pure/puretest"
)

// pure is the model tier's — AUTO-PURE-DETERMINISTIC states it — so the suite
// generates one smoke call and nothing else: no context to cancel, no error to
// carry a zero beside.
//
// Two subjects rather than one, because purity is a property of the method and
// not of any receiver, so the check written once runs against both.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	puretest.RunMixed(t,
		puretest.MixedHarness[*puretest.InMemory]{Name: "in-memory", New: puretest.NewInMemory},
		puretest.MixedHarness[*puretest.InMemory]{Name: "in-memory, second instance", New: puretest.NewInMemory},
		puretest.MixedChecks{
			{
				Method: "Derive",
				Name:   "agrees-with-itself",
				Claim:  "Derive agrees with itself",
				Run: func(tb testing.TB, s pure.Mixed, fx puretest.MixedFixture) {
					tb.Helper()
					// The whole of the mixin's law, and a claim one call cannot
					// make: nothing was observed between the two, so nothing may
					// differ.
					testkit.Equal(tb, s.Derive(fx.Input()), s.Derive(fx.Input()),
						"repeated calls on one input agree")
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

	puretest.RunMixed(t,
		puretest.MixedHarness[*puretest.InMemory]{Name: "in-memory", New: puretest.NewInMemory},
		puretest.MixedSuite.Without(puretest.MixedSuite.Checks.Derive.Smoke()),
	)
}
