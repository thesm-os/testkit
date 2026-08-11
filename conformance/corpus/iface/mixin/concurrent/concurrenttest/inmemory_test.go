// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package concurrenttest_test

import (
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/concurrent"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/concurrent/concurrenttest"
)

// concurrent is the suite tier's under ADR-0018 and its check is still not
// generated, for the reason the RFC's open list records: concurrent callers not
// racing is observable only under the race detector, and `make check` runs
// `mod`, `lint`, `test`, `coverage` and `branch` — not `test race`.
//
// A generated check asserting nothing under the default gate would read as
// coverage while being decoration, so the claim is made here, where what it
// needs is visible.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	concurrenttest.AssertMixedContract(t,
		concurrenttest.MixedSubject("in-memory", func() concurrent.Mixed {
			return concurrenttest.NewInMemory()
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	concurrenttest.AssertMixedContract(t,
		concurrenttest.MixedSubject("in-memory", func() concurrent.Mixed {
			return concurrenttest.NewInMemory()
		}),
		concurrenttest.MixedWithout("Bump/smoke"),
		concurrenttest.MixedWithoutDouble(),
	)
}
