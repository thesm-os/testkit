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
func TestMixedContract(t *testing.T) {
	t.Parallel()

	xsssafetest.AssertMixedContract(t,
		xsssafetest.MixedSubject("in-memory", func() xsssafe.Mixed {
			return xsssafetest.NewInMemory()
		}),
		// Dropped rather than satisfied: escaping is defined for every string, so there is no input Render
		// refuses — the zero-on-error check has no miss to find.
		xsssafetest.MixedWithout("Render/an error carries the zero value"),
		xsssafetest.MixedOnRender("leaves no angle bracket in the output", func(
			tb testing.TB, subject xsssafe.Mixed, in string,
		) {
			tb.Helper()
			// The derived value is well-formed, so the hostile input is
			// supplied here — a check handed only benign text would pass
			// against a subject that escapes nothing.
			got, err := subject.Render(tb.Context(), `<script>alert(1)</script>`)
			testkit.NoError(tb, err, "rendering succeeds")
			testkit.NotContains(tb, got, "<", "no bracket survives escaping")
			_ = in
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	xsssafetest.AssertMixedContract(t,
		xsssafetest.MixedSubject("in-memory", func() xsssafe.Mixed {
			return xsssafetest.NewInMemory()
		}),
		// Dropped rather than satisfied: escaping is defined for every string, so there is no input Render
		// refuses — the zero-on-error check has no miss to find.
		xsssafetest.MixedWithout("Render/an error carries the zero value"),
		xsssafetest.MixedWithoutDouble(),
	)
}
