// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package lookuptest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/lookup"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/lookup/lookuptest"
)

// No context and no error leaves exactly one generated check: the smoke call.
//
// That is the floor, and it is still worth having — a method that panics on a
// derived key is one nothing else in the file would reach. Everything else the
// shape means is written here: nothing on this interface writes, so the miss
// the header refuses is one only a seeding consumer can state.
func TestLookupContract(t *testing.T) {
	t.Parallel()

	fx := lookuptest.DefaultLookupFixture()

	lookuptest.RunLookup(t,
		lookuptest.LookupHarness[*lookuptest.InMemory]{
			Name: "in-memory",
			New: func() *lookuptest.InMemory {
				s := lookuptest.NewInMemory()
				s.Put(
					lookup.Value{Key: fx.Key(), Body: "seeded"},
					lookup.Meta{Revision: 3},
				)
				return s
			},
		},
		lookuptest.LookupChecks{
			{
				Method: "Inspect",
				Name:   "both-slots-for-a-hit",
				Claim:  "Inspect returns both slots for a hit",
				Run: func(tb testing.TB, s lookup.Lookup, fx lookuptest.LookupFixture) {
					tb.Helper()
					v, m, ok := s.Inspect(fx.Key())
					testkit.True(tb, ok, "a seeded key is reported present")
					testkit.Equal(tb, v.Body, "seeded", "the value slot carries what was written")
					testkit.Equal(tb, m.Revision, 3, "and the metadata slot agrees")
				},
			},
			{
				Method: "Inspect",
				Name:   "both-slots-zero-on-a-miss",
				Claim:  "Inspect zeroes both slots for a key nothing wrote",
				Run: func(tb testing.TB, s lookup.Lookup, fx lookuptest.LookupFixture) {
					tb.Helper()
					// Both, not one: a subject zeroing the value and leaking
					// the metadata satisfies a check that reads one slot of
					// two.
					v, m, ok := s.Inspect(fx.KeyOther())
					testkit.False(tb, ok, "an unwritten key is reported absent")
					testkit.Equal(tb, v, lookup.Value{}, "the value slot is the zero")
					testkit.Equal(tb, m, lookup.Meta{}, "and so is the metadata slot")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestLookupContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	lookuptest.RunLookup(t,
		lookuptest.LookupHarness[*lookuptest.InMemory]{Name: "in-memory", New: lookuptest.NewInMemory},
		lookuptest.LookupSuite.Without(lookuptest.LookupSuite.Checks.Inspect.Smoke()),
	)
}
