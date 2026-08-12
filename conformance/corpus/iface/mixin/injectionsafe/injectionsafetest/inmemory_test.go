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
		injectionsafetest.MixedModel(),
		injectionsafetest.MixedSubject("in-memory", func() injectionsafe.Mixed {
			return injectionsafetest.NewInMemory()
		}),
		injectionsafetest.MixedOnStore("round-trips a control sequence as data", func(
			tb testing.TB, subject injectionsafe.Mixed, key, value string,
		) {
			tb.Helper()
			const hostile = `'; DROP TABLE users; --`
			testkit.NoError(tb, subject.Store(tb.Context(), key, hostile), "storing succeeds")
			got, err := subject.Load(tb.Context(), key)
			testkit.NoError(tb, err, "loading succeeds")
			testkit.Equal(tb, got, hostile, "the value is data, not syntax")
			_ = value
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
