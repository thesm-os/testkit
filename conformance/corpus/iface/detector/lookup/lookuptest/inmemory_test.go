// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package lookuptest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/lookup"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/lookup/lookuptest"
)

// No context and no error leaves exactly one generated check: the smoke call.
//
// That is the floor, and it is still worth having — a method that panics on a
// derived key is one nothing else in the file would reach. Everything the shape
// means is written here, because a synchronous multi-slot answer with a flag has
// no property a single call can state.
func TestLookupContract(t *testing.T) {
	t.Parallel()

	// The values the run itself uses, read rather than replaced: nothing here
	// passes LookupWithFixture, so the derivation stands.
	fixture := lookuptest.DefaultLookupFixture()

	lookuptest.AssertLookupContract(t,
		lookuptest.LookupModel(),
		lookuptest.LookupSubject("in-memory", func() lookup.Lookup {
			return lookuptest.NewInMemory()
		}),
		lookuptest.LookupSeed(func(_ context.Context, subject lookup.Lookup) error {
			// A seed may reach for the concrete subject: it runs before the
			// double wraps it and sees what the factory made. A check may not.
			subject.(*lookuptest.InMemory).Put(
				lookup.Value{Key: fixture.Key, Body: "seeded"},
				lookup.Meta{Revision: 3},
			)
			return nil
		}),
		lookuptest.LookupOnInspect("returns both slots for a hit", func(
			tb testing.TB, subject lookup.Lookup, key string,
		) {
			tb.Helper()
			// Only the hit. That a miss zeroes *both* slots is the lookup
			// classification's own check and is generated — and it is the half
			// worth generating, since a subject zeroing the value and leaking
			// the metadata satisfies a check that reads one slot of two.
			v, m, ok := subject.Inspect(key)
			testkit.True(tb, ok, "a seeded key is reported present")
			testkit.Equal(tb, v.Body, "seeded", "the value slot carries what was written")
			testkit.Equal(tb, m.Revision, 3, "and the metadata slot agrees")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestLookupContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	lookuptest.AssertLookupContract(t,
		lookuptest.LookupSubject("in-memory", func() lookup.Lookup {
			return lookuptest.NewInMemory()
		}),
		lookuptest.LookupWithout("Inspect/smoke"),
		lookuptest.LookupWithoutDouble(),
	)
}
