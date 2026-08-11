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

	puretest.AssertMixedContract(t,
		puretest.MixedSubject("in-memory", func() pure.Mixed {
			return puretest.NewInMemory()
		}),
		puretest.MixedSubject("in-memory, second instance", func() pure.Mixed {
			return puretest.NewInMemory()
		}),
		puretest.MixedOnDerive("agrees with itself", func(
			tb testing.TB, subject pure.Mixed, input string,
		) {
			tb.Helper()
			// The whole of the mixin's law, and a claim one call cannot make:
			// nothing was observed between the two, so nothing may differ.
			testkit.Equal(tb, subject.Derive(input), subject.Derive(input),
				"repeated calls on one input agree")
		}),
	)
}

// Two receivers agree, which is stronger than one receiver agreeing with
// itself: a subject caching its last answer satisfies the check above and
// fails this.
func TestDeriveDependsOnItsInputAlone(t *testing.T) {
	t.Parallel()

	first, second := puretest.NewInMemory(), puretest.NewInMemory()
	testkit.Equal(t, first.Derive("x"), second.Derive("x"),
		"two receivers derive the same value from one input")
	testkit.False(t, first.Derive("x") == first.Derive("y"),
		"and different inputs derive differently, or the method is a constant")
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	puretest.AssertMixedContract(t,
		puretest.MixedSubject("in-memory", func() pure.Mixed {
			return puretest.NewInMemory()
		}),
		puretest.MixedWithout("Derive/smoke"),
		puretest.MixedWithoutDouble(),
	)
}
