// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package voidlifecycletest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/voidlifecycle"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/voidlifecycle/voidlifecycletest"
)

// A teardown that cannot fail earns one check, and it is the right one.
//
// With no error return and no context, the only way to get Stop wrong is to
// crash — which is exactly what the smoke call catches. Idempotence is a claim
// about two calls, so it is written here.
func TestVoidLifecycleContract(t *testing.T) {
	t.Parallel()

	voidlifecycletest.AssertVoidLifecycleContract(t,
		voidlifecycletest.VoidLifecycleSubject("in-memory", func() voidlifecycle.VoidLifecycle {
			return voidlifecycletest.NewInMemory()
		}),
		voidlifecycletest.VoidLifecycleOnStop("is idempotent", func(
			tb testing.TB, subject voidlifecycle.VoidLifecycle,
		) {
			tb.Helper()
			// All that is left of the lifecycle law once the error return goes:
			// a second Stop must be safe, and safe is all it can be, since
			// there is nothing to report.
			subject.Stop()
			subject.Stop()
		}),
	)
}

// The effect is out of band, so observing teardown needs a method the interface
// does not declare.
func TestStopIsObservable(t *testing.T) {
	t.Parallel()

	s := voidlifecycletest.NewInMemory()
	testkit.False(t, s.Stopped(), "a fresh subject is running")
	s.Stop()
	testkit.True(t, s.Stopped(), "and stopping it is observable")
	s.Stop()
	testkit.True(t, s.Stopped(), "and stays so on a second call")
}

// Declining the double is separate from dropping a check.
func TestVoidLifecycleContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	voidlifecycletest.AssertVoidLifecycleContract(t,
		voidlifecycletest.VoidLifecycleSubject("in-memory", func() voidlifecycle.VoidLifecycle {
			return voidlifecycletest.NewInMemory()
		}),
		voidlifecycletest.VoidLifecycleWithout("Stop/smoke"),
		voidlifecycletest.VoidLifecycleWithoutDouble(),
	)
}
