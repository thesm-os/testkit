// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package eventuallytest_test

import (
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/eventually"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/eventually/eventuallytest"
)

// eventually is the model tier's — AUTO-EVENTUAL-CONVERGENCE states it — so the
// suite generates the signature family alone.
//
// eidos held it out of the relational set deliberately: "observable eventually"
// raises *observable by what*, the same question `sideeffect` raises, and
// whether it joins that vocabulary is an open call. This fixture answers it
// structurally instead — Settle is the seam, so convergence is driven rather
// than waited for, and no clock is involved.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	eventuallytest.AssertMixedContract(t,
		eventuallytest.MixedSubject("in-memory", func() eventually.Mixed {
			return eventuallytest.NewInMemory()
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	eventuallytest.AssertMixedContract(t,
		eventuallytest.MixedSubject("in-memory", func() eventually.Mixed {
			return eventuallytest.NewInMemory()
		}),
		eventuallytest.MixedWithout("Publish/smoke"),
		eventuallytest.MixedWithoutDouble(),
	)
}
