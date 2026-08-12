// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cacheabletest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/cacheable"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/cacheable/cacheabletest"
)

// cacheable is the model tier's — AUTO-CACHEABLE states it — so the suite
// generates the signature family alone.
//
// The assignment is the right one here for a reason visible in the subject: a
// cached read and an uncached one return the same value, so no single call can
// tell them apart. Observing it needs a reference to compare against, which is
// what the model tier has and this one does not.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	fixture := cacheabletest.DefaultMixedFixture()

	cacheabletest.AssertMixedContract(t,
		cacheabletest.MixedModel(),
		cacheabletest.MixedSubject("in-memory", func() cacheable.Mixed {
			return cacheabletest.NewInMemory()
		}),
		cacheabletest.MixedSeed(func(_ context.Context, subject cacheable.Mixed) error {
			subject.(*cacheabletest.InMemory).Put(fixture.Key, "cached")
			return nil
		}),
		cacheabletest.MixedOnGet("returns what was seeded", func(
			tb testing.TB, subject cacheable.Mixed, key string,
		) {
			tb.Helper()
			got, err := subject.Get(tb.Context(), key)
			testkit.NoError(tb, err, "a seeded key is found")
			testkit.Equal(tb, got, "cached", "and carries what was written")
		}),
		cacheabletest.MixedOnGet("answers a repeated read from the cache", func(
			tb testing.TB, subject cacheable.Mixed, key string,
		) {
			tb.Helper()
			// The mixin's own shape: the second read must agree with the first
			// without consulting what backs it. Both answers look alike, which
			// is why one read cannot state this and two can.
			first, err := subject.Get(tb.Context(), key)
			testkit.NoError(tb, err, "the first read finds the seeded key")
			second, err := subject.Get(tb.Context(), key)
			testkit.NoError(tb, err, "and so does the second")
			testkit.Equal(tb, second, first, "with the same answer")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	cacheabletest.AssertMixedContract(t,
		cacheabletest.MixedSubject("in-memory", func() cacheable.Mixed {
			return cacheabletest.NewInMemory()
		}),
		cacheabletest.MixedWithout("Get/smoke"),
		cacheabletest.MixedWithoutDouble(),
	)
}
