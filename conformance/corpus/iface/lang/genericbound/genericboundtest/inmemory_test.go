// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package genericboundtest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/genericbound"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/genericbound/genericboundtest"
)

// A constraint narrower than `any` changes nothing about the harness: the
// consumer instantiates at a type the constraint admits, and the compiler
// refuses one it does not.
//
// The instantiation here is the pair the source pins through
// `//testkit:stub witness=int,Score`, so the harness, the double and the
// double's own generated checks all run at one set of types.
func TestRankedContract(t *testing.T) {
	t.Parallel()

	fixture := genericboundtest.DefaultRankedFixture[int, genericbound.Score]()
	fixture.Key = 7
	fixture.KeyOther = 9

	genericboundtest.AssertRankedContract[int, genericbound.Score](t,
		genericboundtest.RankedSubject[int, genericbound.Score]("in-memory",
			func() genericbound.Ranked[int, genericbound.Score] {
				return genericboundtest.NewInMemory[int, genericbound.Score]()
			}),
		// A type parameter admits no literal, so the key and the value are the
		// consumer's to name. This is exactly what the option exists for.
		genericboundtest.RankedWithFixture[int, genericbound.Score](fixture),
		genericboundtest.RankedSeed[int, genericbound.Score](
			func(_ context.Context, subject genericbound.Ranked[int, genericbound.Score]) error {
				// A seed may reach for the concrete subject: it runs before the
				// double wraps it and sees what the factory made. A check may
				// not — it runs against every subject the suite is given.
				s := subject.(*genericboundtest.InMemory[int, genericbound.Score])
				s.Set(fixture.Key, genericbound.Score{Points: 1})
				return nil
			}),
		genericboundtest.RankedOnRank[int, genericbound.Score]("returns what was set",
			func(tb testing.TB, subject genericbound.Ranked[int, genericbound.Score], key int) {
				tb.Helper()
				got, err := subject.Rank(tb.Context(), key)
				testkit.NoError(tb, err, "a set key is ranked")
				testkit.Equal(tb, got, genericbound.Score{Points: 1},
					"and carries what was written")
			}),
	)
}

// Declining the double is separate from dropping a check, and a consumer who
// does not use the double should not pay for a second pass over every check.
func TestRankedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	genericboundtest.AssertRankedContract[int, genericbound.Score](t,
		genericboundtest.RankedSubject[int, genericbound.Score]("in-memory",
			func() genericbound.Ranked[int, genericbound.Score] {
				return genericboundtest.NewInMemory[int, genericbound.Score]()
			}),
		genericboundtest.RankedWithout[int, genericbound.Score]("Reset/smoke"),
		genericboundtest.RankedWithoutDouble[int, genericbound.Score](),
	)
}

// Reset empties the ranking rather than only returning nil, which the contract
// cannot state: observing it needs Set, and Set is not on the interface.
func TestResetEmpties(t *testing.T) {
	t.Parallel()

	s := genericboundtest.NewInMemory[int, genericbound.Score]()
	s.Set(7, genericbound.Score{Points: 1})
	testkit.NoError(t, s.Reset(t.Context()), "resetting an open ranking succeeds")
	_, err := s.Rank(t.Context(), 7)
	testkit.ErrorIs(t, err, genericboundtest.ErrNotFound, "and the ranking is gone")
}
