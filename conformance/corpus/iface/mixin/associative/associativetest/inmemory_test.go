// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package associativetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/associative"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/associative/associativetest"
)

// The generated contract, run against the in-memory subject.
//
// Apply and Total are separate methods and nothing in either signature pairs
// them, so that a fold shows up in the total is the row's claim to make.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	associativetest.RunMixed(t,
		associativetest.MixedHarness[*associativetest.InMemory]{Name: "in-memory", New: associativetest.NewInMemory},
		associativetest.MixedChecks{
			{
				Method: "Total",
				Name:   "reflects-what-apply-folded",
				Claim:  "Total reports what Apply folded in",
				Run: func(tb testing.TB, s associative.Mixed, fx associativetest.MixedFixture) {
					tb.Helper()
					// The row applies the delta itself: nothing seeds a subject
					// now but its own constructor, and a total of zero against
					// an untouched fold would state nothing.
					testkit.NoError(tb, s.Apply(tb.Context(), fx.Delta()), "the delta is folded in")

					got, err := s.Total(tb.Context())
					testkit.NoError(tb, err, "the total is readable")
					testkit.NotEqual(tb, got, 0, "and reflects the applied delta")
				},
			},
		},
	)
}
