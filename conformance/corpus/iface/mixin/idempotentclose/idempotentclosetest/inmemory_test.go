// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package idempotentclosetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/idempotentclose"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/idempotentclose/idempotentclosetest"
)

// idempotent-on-a-teardown is the model tier's under ADR-0018:
// `AUTO-IDEMPOTENT-LIFECYCLE` states it, reading Stats across the second
// close.
//
// The check below is the deterministic complement: the exact double-close a
// deferred cleanup performs, asserted by hand. The generated Close/idempotent
// asks only that the second call not error; this one reads the state either
// side of it.
func TestCloserContract(t *testing.T) {
	t.Parallel()

	idempotentclosetest.RunCloser(
		t,
		idempotentclosetest.CloserHarness[*idempotentclosetest.InMemory]{
			Name: "in-memory",
			New:  idempotentclosetest.NewInMemory,
		},
		idempotentclosetest.CloserChecks{
			{
				Method: "Close",
				Name:   "second-close-changes-nothing",
				Claim:  "Close leaves the same state the first one did",
				Run: func(tb testing.TB, s idempotentclose.Closer, fx idempotentclosetest.CloserFixture) {
					tb.Helper()
					testkit.NoError(tb, s.Close(tb.Context()), "the teardown runs")
					open, err := s.Stats(tb.Context())
					testkit.NoError(tb, err, "the state is readable after it")
					testkit.Equal(tb, open, 0, "and nothing is left open")

					testkit.NoError(tb, s.Close(tb.Context()), "closing again is silent")
					again, err := s.Stats(tb.Context())
					testkit.NoError(tb, err, "the state is still readable")
					testkit.Equal(tb, again, 0, "and still nothing is open")
				},
			},
		},
	)
}
