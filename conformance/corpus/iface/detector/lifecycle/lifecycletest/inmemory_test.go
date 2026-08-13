// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package lifecycletest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/lifecycle"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/lifecycle/lifecycletest"
)

// A failable teardown earns the context family and nothing else: Close returns
// an error alone, so there is no value to hold to the zero.
//
// Everything the shape is actually about — that Close is idempotent, that later
// operations report the sentinel — is a law over two calls or two methods, which
// no single-call check states. The first is written here; the second needs a
// method the interface does not declare, and is below.
func TestLifecycleContract(t *testing.T) {
	t.Parallel()

	lifecycletest.AssertLifecycleContract(t,
		lifecycletest.LifecycleModel(),
		lifecycletest.LifecycleSubject("in-memory", func() lifecycle.Lifecycle {
			return lifecycletest.NewInMemory()
		}),
		lifecycletest.LifecycleOnClose("is idempotent", func(
			tb testing.TB, subject lifecycle.Lifecycle,
		) {
			tb.Helper()
			testkit.NoError(tb, subject.Close(tb.Context()), "the first close succeeds")
			testkit.NoError(tb, subject.Close(tb.Context()), "and so does the second")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestLifecycleContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	lifecycletest.AssertLifecycleContract(t,
		lifecycletest.LifecycleSubject("in-memory", func() lifecycle.Lifecycle {
			return lifecycletest.NewInMemory()
		}),
		lifecycletest.LifecycleWithout("Close/smoke"),
		lifecycletest.LifecycleWithoutDouble(),
	)
}

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestLifecycleSaturation(t *testing.T) {
	t.Parallel()
	lifecycletest.LifecycleModelSaturation(t, func() lifecycle.Lifecycle {
		return lifecycletest.NewInMemory()
	})
}
