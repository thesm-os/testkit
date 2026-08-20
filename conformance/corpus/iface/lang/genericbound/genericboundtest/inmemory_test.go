// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package genericboundtest_test

import (
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

	genericboundtest.RunRanked[int, genericbound.Score](t,
		genericboundtest.RankedHarness[int, genericbound.Score, *genericboundtest.InMemory[int, genericbound.Score]]{
			Name: "in-memory",
			// The seed is the constructor's, and it names the key it stores —
			// a type parameter admits no literal, so nothing derived could.
			New: func() *genericboundtest.InMemory[int, genericbound.Score] {
				s := genericboundtest.NewInMemory[int, genericbound.Score]()
				s.Set(7, genericbound.Score{Points: 1})
				return s
			},
		},
		genericboundtest.RankedChecks[int, genericbound.Score]{
			{
				Method: "Rank",
				Name:   "returns-what-was-set",
				Claim:  "Rank returns what was set",
				Run: func(tb testing.TB, s genericbound.Ranked[int, genericbound.Score], fx genericboundtest.RankedFixture[int, genericbound.Score]) {
					tb.Helper()
					got, err := s.Rank(tb.Context(), 7)
					testkit.NoError(tb, err, "a set key is ranked")
					testkit.Equal(tb, got, genericbound.Score{Points: 1},
						"and carries what was written")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestRankedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	genericboundtest.RunRanked[int, genericbound.Score](t,
		genericboundtest.RankedHarness[int, genericbound.Score, *genericboundtest.InMemory[int, genericbound.Score]]{
			Name: "in-memory",
			New:  genericboundtest.NewInMemory[int, genericbound.Score],
		},
		genericboundtest.RankedWithout[int, genericbound.Score](genericboundtest.RankedSuite.Checks.Reset.Smoke()),
	)
}
