// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package xsssafetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/xsssafe"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/xsssafe/xsssafetest"
)

// The generated contract, run against the in-memory subject.
//
// Render is a transform: nothing writes and nothing seeds, so the reader
// shape's miss check was refused rather than emitted — the header records it.
// What escaping owes is the row's, because the derived value is well-formed
// and a check handed only benign text would pass against a subject that
// escapes nothing.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	xsssafetest.RunMixed(t,
		xsssafetest.MixedHarness[*xsssafetest.InMemory]{Name: "in-memory", New: xsssafetest.NewInMemory},
		xsssafetest.MixedChecks{
			{
				Method: "Render",
				Name:   "leaves-no-angle-bracket",
				Claim:  "Render leaves no angle bracket in the output",
				Run: func(tb testing.TB, s xsssafe.Mixed, fx xsssafetest.MixedFixture) {
					tb.Helper()
					got, err := s.Render(tb.Context(), `<script>alert(1)</script>`)
					testkit.NoError(tb, err, "rendering succeeds")
					testkit.NotContains(tb, got, "<", "no bracket survives escaping")
				},
			},
		},
	)
}
