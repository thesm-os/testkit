// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cacheabletest_test

import (
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
//
// Nothing on this interface writes, so the reader shape's miss check was
// refused; the seed lives in the constructor instead, which is where a seeded
// subject is built now.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	fx := cacheabletest.DefaultMixedFixture()

	cacheabletest.RunMixed(t,
		cacheabletest.MixedHarness[*cacheabletest.InMemory]{
			Name: "in-memory",
			New: func() *cacheabletest.InMemory {
				s := cacheabletest.NewInMemory()
				s.Put(fx.Key(), "cached")
				return s
			},
		},
		cacheabletest.MixedChecks{
			{
				Method: "Get",
				Name:   "returns-what-was-seeded",
				Claim:  "Get returns what was seeded",
				Run: func(tb testing.TB, s cacheable.Mixed, fx cacheabletest.MixedFixture) {
					tb.Helper()
					got, err := s.Get(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "a seeded key is found")
					testkit.Equal(tb, got, "cached", "and carries what was written")
				},
			},
			{
				Method: "Get",
				Name:   "repeated-read-agrees",
				Claim:  "Get answers a repeated read from the cache",
				Run: func(tb testing.TB, s cacheable.Mixed, fx cacheabletest.MixedFixture) {
					tb.Helper()
					// The mixin's own shape: the second read must agree with
					// the first without consulting what backs it. Both answers
					// look alike, which is why one read cannot state this and
					// two can.
					first, err := s.Get(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "the first read finds the seeded key")
					second, err := s.Get(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "and so does the second")
					testkit.Equal(tb, second, first, "with the same answer")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	cacheabletest.RunMixed(t,
		cacheabletest.MixedHarness[*cacheabletest.InMemory]{Name: "in-memory", New: cacheabletest.NewInMemory},
		cacheabletest.MixedSuite.Without(cacheabletest.MixedSuite.Checks.Get.Smoke()),
	)
}
