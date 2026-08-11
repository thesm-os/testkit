// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package injectionsafetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/injectionsafe"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/injectionsafe/injectionsafetest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	injectionsafetest.AssertMixedContract(t,
		injectionsafetest.MixedSubject("in-memory", func() injectionsafe.Mixed {
			return injectionsafetest.NewInMemory()
		}),
		// Dropped rather than satisfied: storing is defined for every string, so there is no input Store
		// refuses — the zero-on-error check has no miss to find.
		injectionsafetest.MixedWithout("Store/an error carries the zero value"),
		injectionsafetest.MixedOnStore("returns a control sequence unchanged", func(
			tb testing.TB, subject injectionsafe.Mixed, in string,
		) {
			tb.Helper()
			const hostile = `'; DROP TABLE users; --`
			got, err := subject.Store(tb.Context(), hostile)
			testkit.NoError(tb, err, "storing succeeds")
			testkit.Equal(tb, got, hostile, "the value is data, not syntax")
			_ = in
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	injectionsafetest.AssertMixedContract(t,
		injectionsafetest.MixedSubject("in-memory", func() injectionsafe.Mixed {
			return injectionsafetest.NewInMemory()
		}),
		// Dropped rather than satisfied: storing is defined for every string, so there is no input Store
		// refuses — the zero-on-error check has no miss to find.
		injectionsafetest.MixedWithout("Store/an error carries the zero value"),
		injectionsafetest.MixedWithoutDouble(),
	)
}
