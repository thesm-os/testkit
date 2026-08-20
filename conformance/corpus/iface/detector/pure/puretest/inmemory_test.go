// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package puretest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/pure"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/pure/puretest"
)

// Two subjects rather than one, which is what a conformance suite is for.
//
// Purity is a property of the method, not of any particular receiver, so both
// implementations answer to one statement of the contract — and a check written
// once runs against each. The generated half is a single smoke call; the law the
// shape carries is that repeated calls agree, and that needs two of them.
func TestPureContract(t *testing.T) {
	t.Parallel()

	puretest.RunPure(t,
		puretest.PureHarness[*puretest.InMemory]{
			Name: "in-memory",
			New:  func() *puretest.InMemory { return puretest.NewInMemory("first") },
		},
		puretest.PureHarness[*puretest.InMemory]{
			Name: "in-memory, relabelled",
			New:  func() *puretest.InMemory { return puretest.NewInMemory("second") },
		},
		puretest.PureChecks{
			{
				Method: "Describe",
				Name:   "agrees-with-itself",
				Claim:  "Describe agrees with itself",
				Run: func(tb testing.TB, s pure.Pure, fx puretest.PureFixture) {
					tb.Helper()
					// The whole of the shape's law: nothing was observed
					// between the two calls, so nothing may differ between the
					// two answers.
					testkit.Equal(tb, s.Describe(), s.Describe(),
						"repeated calls derive the same value from the same receiver")
				},
			},
			{
				Method: "Describe",
				Name:   "says-something",
				Claim:  "Describe says something",
				Run: func(tb testing.TB, s pure.Pure, fx puretest.PureFixture) {
					tb.Helper()
					// Otherwise the check above is satisfied by returning "".
					testkit.False(tb, s.Describe() == "",
						"a description that is empty agrees with itself and says nothing")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestPureContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	puretest.RunPure(t,
		puretest.PureHarness[*puretest.InMemory]{
			Name: "in-memory",
			New:  func() *puretest.InMemory { return puretest.NewInMemory("first") },
		},
		puretest.PureSuite.Without(puretest.PureSuite.Checks.Describe.Smoke()),
	)
}
