// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package concurrentreaderstest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
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
