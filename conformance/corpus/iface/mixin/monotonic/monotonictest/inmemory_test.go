// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package monotonictest_test

import (
	"testing"

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
		monotonictest.MixedModel(),
		monotonictest.MixedSubject("in-memory", func() monotonic.Mixed {
			return monotonictest.NewInMemory()
		}),
	)
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

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestMixedSaturation(t *testing.T) {
	t.Parallel()
	monotonictest.MixedModelSaturation(t, func() monotonic.Mixed {
		return monotonictest.NewInMemory()
	})
}
