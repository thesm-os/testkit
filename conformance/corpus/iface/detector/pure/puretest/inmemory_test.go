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

	puretest.AssertPureContract(t,
		puretest.PureSubject("in-memory", func() pure.Pure {
			return puretest.NewInMemory("first")
		}),
		puretest.PureSubject("in-memory, relabelled", func() pure.Pure {
			return puretest.NewInMemory("second")
		}),
		puretest.PureOnDescribe("agrees with itself", func(
			tb testing.TB, subject pure.Pure,
		) {
			tb.Helper()
			// The whole of the shape's law: nothing was observed between the
			// two calls, so nothing may differ between the two answers.
			testkit.Equal(tb, subject.Describe(), subject.Describe(),
				"repeated calls derive the same value from the same receiver")
		}),
		puretest.PureOnDescribe("says something", func(
			tb testing.TB, subject pure.Pure,
		) {
			tb.Helper()
			// Otherwise the check above is satisfied by returning "".
			testkit.False(tb, subject.Describe() == "",
				"a description that is empty agrees with itself and says nothing")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestPureContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	puretest.AssertPureContract(t,
		puretest.PureSubject("in-memory", func() pure.Pure {
			return puretest.NewInMemory("first")
		}),
		puretest.PureWithout("Describe/smoke"),
		puretest.PureWithoutDouble(),
	)
}
