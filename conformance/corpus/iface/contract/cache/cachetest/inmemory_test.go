// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cachetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/cache"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/cache/cachetest"
)

// cache is the model tier's under ADR-0018: `AUTO-CACHEABLE` states it.
//
// Neither role writes, so nothing is derived to seed through and the header
// refuses both miss checks. The seeding is the constructor's, which is where a
// seeded subject is built now — a factory may make any starting state, and it
// runs before anything wraps it.
func TestContractContract(t *testing.T) {
	t.Parallel()

	fx := cachetest.DefaultContractFixture()

	cachetest.RunContract(t,
		cachetest.ContractHarness[*cachetest.InMemory]{
			Name: "in-memory",
			New: func() *cachetest.InMemory {
				s := cachetest.NewInMemory()
				s.Store(cache.Value{Key: fx.Key(), Body: "seeded"})
				return s
			},
		},
		cachetest.ContractHarness[*cachetest.InMemory]{
			Name: "in-memory, already warmed",
			// The cached read is invisible through the interface — both reads
			// answer the same thing whether or not anything was cached — so it
			// is reached by handing the run a subject whose cache is already
			// warm and whose backing no longer holds the key. Every read
			// against this one is a hit, and a subject with no cache misses.
			New: func() *cachetest.InMemory {
				s := cachetest.NewInMemory()
				s.Store(cache.Value{Key: fx.Key(), Body: "seeded"})
				if _, err := s.Lookup(t.Context(), fx.Key()); err != nil {
					panic("cachetest_test: warming the cache: " + err.Error())
				}
				s.Forget(fx.Key())
				return s
			},
		},
		cachetest.ContractChecks{
			{
				Method: "Lookup",
				Name:   "answers-from-the-backing-store",
				Claim:  "Lookup answers from the backing store on a miss",
				Run: func(tb testing.TB, s cache.Contract, fx cachetest.ContractFixture) {
					tb.Helper()
					got, err := s.Lookup(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "a seeded key is found")
					testkit.Equal(tb, got.Body, "seeded", "and carries what was stored")
				},
			},
			{
				Method: "Lookup",
				Name:   "misses-a-key-nothing-holds",
				Claim:  "Lookup reports a key neither role holds",
				Run: func(tb testing.TB, s cache.Contract, fx cachetest.ContractFixture) {
					tb.Helper()
					// The claim the generator refuses here — nothing on this
					// interface writes, so it cannot tell an unheld key from
					// any other. The constructor above is what makes the
					// distinction real.
					_, err := s.Lookup(tb.Context(), fx.KeyOther())
					testkit.ErrorIs(tb, err, cachetest.ErrNotFound,
						"a key nothing seeded is a miss rather than an unlabelled failure")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	cachetest.RunContract(t,
		cachetest.ContractHarness[*cachetest.InMemory]{Name: "in-memory", New: cachetest.NewInMemory},
		cachetest.ContractSuite.Without(cachetest.ContractSuite.Checks.Lookup.Smoke()),
	)
}
