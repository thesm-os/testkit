// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package concurrenttest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/concurrency"
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

// Eight callers bumping a hundred times each lose no updates. A subject reading
// and writing outside a lock passes every generated check — the call succeeds
// and Count still answers — and arrives at a total lower than the calls made.
func TestConcurrentBumpsLoseNothing(t *testing.T) {
	t.Parallel()

	s := concurrenttest.NewInMemory()
	ctx := t.Context()

	defer concurrency.GoroutineLeak(t)()
	concurrency.ConcurrentStress(t, 8, 100, func(g, _ int) {
		testkit.NoError(t, s.Bump(ctx, string(rune('a'+g))), "a concurrent bump succeeds")
	})

	got, err := s.Count(ctx)
	testkit.NoError(t, err, "counting succeeds")
	testkit.Equal(t, got, 800, "every bump landed")
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
