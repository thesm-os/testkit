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
	)
}

// The builder samples rather than returning one fixed member of the space, or
// a suite running it a thousand times exercises one input a thousand times.
func TestNewInputSamplesRatherThanRepeats(t *testing.T) {
	t.Parallel()

	s := sampletest.NewInMemory()
	first, err := s.NewInput(t.Context())
	testkit.NoError(t, err, "the first sample is produced")
	second, err := s.NewInput(t.Context())
	testkit.NoError(t, err, "and so is the second")
	testkit.False(t, first == second, "consecutive samples differ")
}

// The refusal is real, which is what the generated check rests on.
func TestProcessRefusesAnUnsampledInput(t *testing.T) {
	t.Parallel()

	s := sampletest.NewInMemory()
	_, err := s.Process(t.Context(), "test-input")
	testkit.ErrorIs(t, err, sampletest.ErrUnprocessable,
		"a value the builder did not produce is refused")
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
