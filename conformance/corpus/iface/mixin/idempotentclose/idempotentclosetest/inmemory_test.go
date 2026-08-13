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
// deferred cleanup performs, asserted by hand.
func TestCloserContract(t *testing.T) {
	t.Parallel()

	idempotentclosetest.AssertCloserContract(t,
		idempotentclosetest.CloserModel(),
		idempotentclosetest.CloserSubject("in-memory", func() idempotentclose.Closer {
			return idempotentclosetest.NewInMemory()
		}),
		idempotentclosetest.CloserOnClose("a second close changes nothing", func(
			tb testing.TB, subject idempotentclose.Closer,
		) {
			tb.Helper()
			testkit.NoError(tb, subject.Close(tb.Context()), "the teardown runs")
			open, err := subject.Stats(tb.Context())
			testkit.NoError(tb, err, "the state is readable after it")
			testkit.Equal(tb, open, 0, "and nothing is left open")

			testkit.NoError(tb, subject.Close(tb.Context()), "closing again is silent")
			again, err := subject.Stats(tb.Context())
			testkit.NoError(tb, err, "the state is still readable")
			testkit.Equal(tb, again, 0, "and still nothing is open")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestCloserContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	idempotentclosetest.AssertCloserContract(t,
		idempotentclosetest.CloserSubject("in-memory", func() idempotentclose.Closer {
			return idempotentclosetest.NewInMemory()
		}),
		idempotentclosetest.CloserWithoutDouble(),
	)
}

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestCloserSaturation(t *testing.T) {
	t.Parallel()
	idempotentclosetest.CloserModelSaturation(t, func() idempotentclose.Closer {
		return idempotentclosetest.NewInMemory()
	})
}
