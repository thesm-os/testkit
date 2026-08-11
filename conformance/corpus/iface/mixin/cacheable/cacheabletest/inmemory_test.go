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
	)
}

// The cache is real, which the interface hides: both reads answer alike, and
// only the miss count says the second was served without reaching the store.
func TestRepeatReadsAreServedFromTheCache(t *testing.T) {
	t.Parallel()

	s := cacheabletest.NewInMemory()
	s.Put("k", "v")

	first, err := s.Get(t.Context(), "k")
	testkit.NoError(t, err, "the first read succeeds")
	second, err := s.Get(t.Context(), "k")
	testkit.NoError(t, err, "and so does the second")

	testkit.Equal(t, second, first, "both answers agree")
	testkit.Equal(t, s.Misses(), 1, "and only the first reached the store")
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
