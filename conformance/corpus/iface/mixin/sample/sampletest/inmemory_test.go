// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sampletest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/sample"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/sample/sampletest"
)

// The only generated check that takes no derived input, because the input is
// the point.
//
// A sampled suite is exactly as good as its sampler, and a builder that drifts
// from what the method accepts turns every run into a wall of failures about
// the fixture rather than the subject. So the check calls the builder the mixin
// names and feeds the result to the method — handing it a value the fixture
// invented would test the derivation instead of the pair.
//
// Process refuses anything without the builder's own prefix, which is what
// makes the check able to fail: a Process accepting every string would be
// satisfied by any builder at all.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	sampletest.AssertMixedContract(t,
		sampletest.MixedSubject("in-memory", func() sample.Mixed {
			return sampletest.NewInMemory()
		}),
		// Process rejects the derived input, which is the fixture working
		// rather than failing: a value the generator invents is precisely what
		// a sampled method does not accept, and is why the mixin names a
		// builder. Both checks below reach Process through NewInput instead.
		sampletest.MixedWithout(
			"Process/smoke",
			"Process/an error carries the zero value",
		),
		sampletest.MixedOnProcess("refuses an input the builder did not produce", func(
			tb testing.TB, subject sample.Mixed, input string,
		) {
			tb.Helper()
			// The constraint is what makes the mixin worth having: a Process
			// accepting anything would pair with any builder, and the check
			// that they fit would hold for a builder producing nonsense.
			_, err := subject.Process(tb.Context(), "unshaped-"+input)
			testkit.Error(tb, err, "an input outside the shape is refused")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	sampletest.AssertMixedContract(t,
		sampletest.MixedSubject("in-memory", func() sample.Mixed {
			return sampletest.NewInMemory()
		}),
		sampletest.MixedWithout(
			"Process/smoke",
			"Process/an error carries the zero value",
		),
		sampletest.MixedWithoutDouble(),
	)
}
