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
//
// The hostile value is the row's, not the fixture's: the derivation writes
// plausible values and a control sequence is exactly what it will not invent.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	injectionsafetest.RunMixed(
		t,
		injectionsafetest.MixedHarness[*injectionsafetest.InMemory]{
			Name: "in-memory",
			New:  injectionsafetest.NewInMemory,
		},
		injectionsafetest.MixedChecks{
			{
				Method: "Store",
				Name:   "control-sequence-is-data",
				Claim:  "Store round-trips a control sequence as data",
				Run: func(tb testing.TB, s injectionsafe.Mixed, fx injectionsafetest.MixedFixture) {
					tb.Helper()
					const hostile = `'; DROP TABLE users; --`
					testkit.NoError(tb, s.Store(tb.Context(), fx.Key(), hostile), "storing succeeds")

					got, err := s.Load(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "loading succeeds")
					testkit.Equal(tb, got, hostile, "the value is data, not syntax")
				},
			},
		},
	)
}
