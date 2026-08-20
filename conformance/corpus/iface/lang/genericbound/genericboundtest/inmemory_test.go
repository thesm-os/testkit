// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// A constraint narrower than `any` changes nothing about the harness: the
// consumer instantiates at a type the constraint admits, and the compiler
// refuses one it does not.
//
// The instantiation here is the pair the source pins through
// `//testkit:stub witness=int,Score`, so the harness, the double and the
// double's own generated checks all run at one set of types.
package genericboundtest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/genericbound"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/genericbound/genericboundtest"
)

// TestRankedContract runs the generated checks and this package's own, at the
// instantiation this package names.
func TestRankedContract(t *testing.T) {
	t.Parallel()

	genericboundtest.RunRanked[int, genericbound.Score](t, inMemory("in-memory"), rankedChecks)
}

// TestRankedContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestRankedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	genericboundtest.RunRanked[int, genericbound.Score](t,
		inMemory("in-memory"),
		genericboundtest.RankedWithout[int, genericbound.Score](
			genericboundtest.RankedSuite.Checks.Reset.Smoke(),
		),
	)
}

// TestRankedChecksCanFail drives the row against its planted defect.
func TestRankedChecksCanFail(t *testing.T) {
	t.Parallel()

	genericboundtest.ProveRanked[int, genericbound.Score](t, rankedChecks)
}

// --- Harnesses ---------------------------------------------------------------

// The key and score the row uses. Literals rather than fixture accessors: a
// type parameter admits no literal, so nothing derived could name either.
const rankedKey = 7

var rankedScore = genericbound.Score{Points: 1}

// inMemory seeds the subject, and the seed names the key it stores.
func inMemory(
	name string,
) genericboundtest.RankedHarness[int, genericbound.Score, *genericboundtest.InMemory[int, genericbound.Score]] {
	return genericboundtest.RankedHarness[
		int, genericbound.Score, *genericboundtest.InMemory[int, genericbound.Score],
	]{Name: name, New: seeded}
}

func seeded() *genericboundtest.InMemory[int, genericbound.Score] {
	s := genericboundtest.NewInMemory[int, genericbound.Score]()
	s.Set(rankedKey, rankedScore)
	return s
}

// --- The checks: claims, bodies and defects, by name --------------------------

var rankedChecks = genericboundtest.RankedChecks[int, genericbound.Score]{
	{
		Method: "Rank", Name: "returns-what-was-set",
		Claim: "Rank returns what was set",
		Run:   returnsWhatWasSet,
		ProvenBy: genericboundtest.BrokenRanked[int, genericbound.Score](
			"a store whose reads answer the value type's zero", newAnswersTheZero,
		),
		ProvenReason: "carries what was written",
	},
}

// --- Bodies -------------------------------------------------------------------

func returnsWhatWasSet(
	tb testing.TB, s genericbound.Ranked[int, genericbound.Score],
	_ genericboundtest.RankedFixture[int, genericbound.Score],
) {
	tb.Helper()
	got, err := s.Rank(tb.Context(), rankedKey)
	testkit.NoError(tb, err, "a set key is ranked")
	testkit.Equal(tb, got, rankedScore, "and carries what was written")
}

// --- Planted defects ----------------------------------------------------------

// answersTheZero finds the key and hands back V's zero, which for a constrained
// type parameter is the same hazard it is for an unconstrained one: the
// constraint bounds what V can BE, and says nothing about which value of it a
// read answers.
type answersTheZero struct{ held map[int]genericbound.Score }

func newAnswersTheZero() *answersTheZero {
	return &answersTheZero{held: map[int]genericbound.Score{rankedKey: rankedScore}}
}

func (a *answersTheZero) Rank(_ context.Context, key int) (genericbound.Score, error) {
	if _, held := a.held[key]; !held {
		return genericbound.Score{}, genericboundtest.ErrNotFound
	}
	return genericbound.Score{}, nil
}

func (a *answersTheZero) Reset(context.Context) error {
	a.held = map[int]genericbound.Score{}
	return nil
}
