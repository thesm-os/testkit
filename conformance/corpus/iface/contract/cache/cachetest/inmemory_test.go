// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cachetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/cache"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/cache/cachetest"
)

// cache is the model tier's under ADR-0018: `AUTO-CACHEABLE` states it.
//
// Neither role writes, so nothing is derived to seed through and the harness
// would run every check against an empty store. The seed hook exists for
// exactly that — and it may reach for the concrete subject, because it runs
// before the double wraps it and sees what the factory made. A check may not.
func TestContractContract(t *testing.T) {
	t.Parallel()

	fixture := cachetest.DefaultContractFixture()

	cachetest.AssertContractContract(t,
		cachetest.ContractModel(),
		cachetest.ContractSubject("in-memory", func() cache.Contract {
			return cachetest.NewInMemory()
		}),
		cachetest.ContractSubject("in-memory, already warmed", func() cache.Contract {
			// The cached read is invisible through the interface — both reads
			// answer the same thing whether or not anything was cached — so it
			// is reached by handing the run a subject whose cache is already
			// warm and whose backing no longer holds the key. Every read
			// against this one is a hit, and a subject with no cache misses.
			//
			// A factory may build any starting state; a check may not, because
			// it receives whatever the factory made and the double wraps it.
			s := cachetest.NewInMemory()
			s.Store(cache.Value{Key: cachetest.DefaultContractFixture().Key, Body: "seeded"})
			if _, err := s.Lookup(t.Context(), cachetest.DefaultContractFixture().Key); err != nil {
				panic("cachetest_test: warming the cache: " + err.Error())
			}
			s.Forget(cachetest.DefaultContractFixture().Key)
			return s
		}),
		cachetest.ContractSeed(func(_ context.Context, subject cache.Contract) error {
			subject.(*cachetest.InMemory).Store(cache.Value{Key: fixture.Key, Body: "seeded"})
			return nil
		}),
		cachetest.ContractOnLookup("answers from the backing store on a miss", func(
			tb testing.TB, subject cache.Contract, key string,
		) {
			tb.Helper()
			got, err := subject.Lookup(tb.Context(), key)
			testkit.NoError(tb, err, "a seeded key is found")
			testkit.Equal(tb, got.Body, "seeded", "and carries what was stored")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	cachetest.AssertContractContract(t,
		cachetest.ContractSubject("in-memory", func() cache.Contract {
			return cachetest.NewInMemory()
		}),
		cachetest.ContractSubject("in-memory, already warmed", func() cache.Contract {
			// The cached read is invisible through the interface — both reads
			// answer the same thing whether or not anything was cached — so it
			// is reached by handing the run a subject whose cache is already
			// warm and whose backing no longer holds the key. Every read
			// against this one is a hit, and a subject with no cache misses.
			//
			// A factory may build any starting state; a check may not, because
			// it receives whatever the factory made and the double wraps it.
			s := cachetest.NewInMemory()
			s.Store(cache.Value{Key: cachetest.DefaultContractFixture().Key, Body: "seeded"})
			if _, err := s.Lookup(t.Context(), cachetest.DefaultContractFixture().Key); err != nil {
				panic("cachetest_test: warming the cache: " + err.Error())
			}
			s.Forget(cachetest.DefaultContractFixture().Key)
			return s
		}),
		cachetest.ContractWithout("Lookup/smoke"),
		cachetest.ContractWithoutDouble(),
	)
}

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestContractSaturation(t *testing.T) {
	t.Parallel()
	cachetest.ContractModelSaturation(t, func() cache.Contract {
		return cachetest.NewInMemory()
	})
}
