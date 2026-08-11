// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package boundedtest_test

import (
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/bounded"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/bounded/boundedtest"
)

// bounded is the model tier's — AUTO-AGGREGATOR-BOUNDED states it — so the
// suite generates the signature family alone.
//
// List takes nothing after its context, so it earns no zero-value check either:
// there is no input to choose that makes it fail.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	boundedtest.AssertMixedContract(t,
		boundedtest.MixedSubject("in-memory", func() bounded.Mixed {
			return boundedtest.NewInMemory()
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	boundedtest.AssertMixedContract(t,
		boundedtest.MixedSubject("in-memory", func() bounded.Mixed {
			return boundedtest.NewInMemory()
		}),
		boundedtest.MixedWithout("List/smoke"),
		boundedtest.MixedWithoutDouble(),
	)
}
