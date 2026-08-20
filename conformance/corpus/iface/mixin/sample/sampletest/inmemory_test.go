// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sampletest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/sample"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/sample/sampletest"
)

// A sampled suite is exactly as good as its sampler, and a builder that drifts
// from what the method accepts turns every run into a wall of failures about
// the fixture rather than the subject.
//
// The mixin has no suite-side rule yet — the header says so — so the pairing
// is written here: reach Process through NewInput, and separately show that
// Process refuses what the builder did not produce. Without the second half a
// Process accepting everything would pair with any builder at all.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	sampletest.RunMixed(t,
		sampletest.MixedHarness[*sampletest.InMemory]{Name: "in-memory", New: sampletest.NewInMemory},
		// Process rejects the derived input, which is the fixture working
		// rather than failing: a value the generator invents is precisely what
		// a sampled method does not accept, and is why the mixin names a
		// builder.
		sampletest.MixedSuite.Without(sampletest.MixedSuite.Checks.Process.Smoke()),
		sampletest.MixedChecks{
			{
				Method: "Process",
				Name:   "accepts-only-what-the-builder-made",
				Claim:  "Process refuses an input the builder did not produce",
				Run: func(tb testing.TB, s sample.Mixed, fx sampletest.MixedFixture) {
					tb.Helper()
					built, err := s.NewInput(tb.Context())
					testkit.NoError(tb, err, "the builder produces an input")

					_, err = s.Process(tb.Context(), built)
					testkit.NoError(tb, err, "which the method accepts")

					// The constraint is what makes the mixin worth having.
					_, err = s.Process(tb.Context(), "unshaped-"+fx.Input())
					testkit.Error(tb, err, "an input outside the shape is refused")
				},
			},
		},
	)
}
