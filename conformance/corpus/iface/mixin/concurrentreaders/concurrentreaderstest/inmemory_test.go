// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package concurrentreaderstest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/concurrency"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/concurrentreaders"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/concurrentreaders/concurrentreaderstest"
)

// concurrentreaders is the suite tier's under ADR-0018, and its check is still
// not generated.
//
// Readers that do not corrupt one another is observable only under the race
// detector, and `make check` runs `mod`, `lint`, `test`, `coverage` and
// `branch` — not `test race`. A generated check asserting nothing under the
// default gate would be decoration that reads as coverage, so the claim is made
// here where its conditions are visible.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	fixture := concurrentreaderstest.DefaultMixedFixture()

	concurrentreaderstest.AssertMixedContract(t,
		concurrentreaderstest.MixedSubject("in-memory", func() concurrentreaders.Mixed {
			return concurrentreaderstest.NewInMemory()
		}),
		concurrentreaderstest.MixedSeed(func(ctx context.Context, subject concurrentreaders.Mixed) error {
			return subject.Put(ctx, fixture.Key, fixture.Value)
		}),
		concurrentreaderstest.MixedOnGet("returns what was written", func(
			tb testing.TB, subject concurrentreaders.Mixed, key string,
		) {
			tb.Helper()
			got, err := subject.Get(tb.Context(), key)
			testkit.NoError(tb, err, "a written key is found")
			testkit.Equal(tb, got, fixture.Value, "and carries what was written")
		}),
	)
}

// Eight readers, a hundred reads each, all seeing one consistent answer.
// Meaningful under `-race`, cheap otherwise — which is the trade that keeps it
// out of the generated harness and in the package that knows how it is run.
//
// Driven through [concurrency.ConcurrentStress] rather than a hand-rolled
// WaitGroup: spawning and joining is not what this test is about, and the
// leak check beside it is a claim a hand-rolled version does not make at all —
// a reader that parked a goroutine per call would pass every assertion here
// and leak one per read.
func TestConcurrentReadsAreSafe(t *testing.T) {
	t.Parallel()

	s := concurrentreaderstest.NewInMemory()
	testkit.NoError(t, s.Put(t.Context(), "k", "v"), "seeding succeeds")

	defer concurrency.GoroutineLeak(t)()

	ctx := t.Context()
	concurrency.ConcurrentStress(t, 8, 100, func(_, _ int) {
		got, err := s.Get(ctx, "k")
		testkit.NoError(t, err, "a concurrent read succeeds")
		testkit.Equal(t, got, "v", "and sees the seeded value")
	})
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	concurrentreaderstest.AssertMixedContract(t,
		concurrentreaderstest.MixedSubject("in-memory", func() concurrentreaders.Mixed {
			return concurrentreaderstest.NewInMemory()
		}),
		concurrentreaderstest.MixedWithout("Put/smoke"),
		concurrentreaderstest.MixedWithoutDouble(),
	)
}
