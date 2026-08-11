// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package monotonictest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/monotonic"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/monotonic/monotonictest"
)

// monotonic is the model tier's — AUTO-MONOTONIC-NON-DECREASING states it — so
// the suite generates the signature family alone.
//
// Version takes nothing after its context, so it earns no zero-value check
// either: there is no input to choose that makes it fail.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	monotonictest.AssertMixedContract(t,
		monotonictest.MixedSubject("in-memory", func() monotonic.Mixed {
			return monotonictest.NewInMemory()
		}),
	)
}

// The version never goes backwards, which no single reading can show: any one
// value is consistent with any sequence that produced it.
func TestVersionNeverDecreases(t *testing.T) {
	t.Parallel()

	s := monotonictest.NewInMemory()
	first, err := s.Version(t.Context())
	testkit.NoError(t, err, "reading the version succeeds")

	testkit.NoError(t, s.Advance(t.Context()), "advancing succeeds")
	second, err := s.Version(t.Context())
	testkit.NoError(t, err, "and reading again succeeds")

	testkit.True(t, second >= first, "the version never moves backwards")
	testkit.True(t, second > first, "and an advance is observable")
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	monotonictest.AssertMixedContract(t,
		monotonictest.MixedSubject("in-memory", func() monotonic.Mixed {
			return monotonictest.NewInMemory()
		}),
		monotonictest.MixedWithout("Version/smoke"),
		monotonictest.MixedWithoutDouble(),
	)
}
