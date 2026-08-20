// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package voidlifecycletest_test

import (
	"testing"

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

	voidlifecycletest.RunVoidLifecycle(
		t,
		voidlifecycletest.VoidLifecycleHarness[*voidlifecycletest.InMemory]{
			Name: "in-memory",
			New:  voidlifecycletest.NewInMemory,
		},
		voidlifecycletest.VoidLifecycleChecks{
			{
				Method: "Stop",
				Name:   "second-stop-is-safe",
				Claim:  "Stop is idempotent",
				Run: func(tb testing.TB, s voidlifecycle.VoidLifecycle, fx voidlifecycletest.VoidLifecycleFixture) {
					tb.Helper()
					// All that is left of the lifecycle law once the error
					// return goes: a second Stop must be safe, and safe is all
					// it can be, since there is nothing to report.
					s.Stop()
					s.Stop()
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestVoidLifecycleContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	voidlifecycletest.RunVoidLifecycle(
		t,
		voidlifecycletest.VoidLifecycleHarness[*voidlifecycletest.InMemory]{
			Name: "in-memory",
			New:  voidlifecycletest.NewInMemory,
		},
		voidlifecycletest.VoidLifecycleSuite.Without(voidlifecycletest.VoidLifecycleSuite.Checks.Stop.Smoke()),
	)
}
