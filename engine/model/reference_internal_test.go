// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal, because what it exercises is internal: the pair and triple
// carriers and the lift functions that raise a single-subject action onto
// them compose multi-subject references inside this package, and nothing
// outside it has ever called them. They were exported, and an external test
// was the only thing keeping that true.

package model

import (
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
)

// --- Minimal types for lift tests ---

type (
	counterA struct{ n int }
	counterB struct{ n int }
	counterC struct{ n int }
)

func incAction(name string) Action[*counterA] {
	return Action[*counterA]{
		Name: name,
		Kind: FailureSemantic,
		Run: func(_ *rapid.T, sut, ref *counterA) ActionResult {
			sut.n++
			ref.n++
			return ActionResult{}
		},
	}
}

func incActionB(name string) Action[*counterB] {
	return Action[*counterB]{
		Name: name,
		Kind: FailureInvariant,
		Run: func(_ *rapid.T, sut, ref *counterB) ActionResult {
			sut.n++
			ref.n++
			return ActionResult{}
		},
	}
}

func incActionC(name string) Action[*counterC] {
	return Action[*counterC]{
		Name: name,
		Run: func(_ *rapid.T, sut, ref *counterC) ActionResult {
			sut.n++
			ref.n++
			return ActionResult{}
		},
	}
}

func TestLiftA(t *testing.T) {
	t.Parallel()
	inner := incAction("Inc")
	lifted := liftA[*counterA, *counterB](inner)

	testkit.Equal(t, lifted.Name, "A.Inc", "liftA prefixes with A.")
	testkit.Equal(t, lifted.Kind, FailureSemantic, "liftA preserves Kind")

	rapid.Check(t, func(rt *rapid.T) {
		sut := pair[*counterA, *counterB]{A: &counterA{}, B: &counterB{}}
		ref := pair[*counterA, *counterB]{A: &counterA{}, B: &counterB{}}
		lifted.Run(rt, sut, ref)
		testkit.Equal(t, sut.A.n, 1, "liftA routes to pair.A SUT")
		testkit.Equal(t, ref.A.n, 1, "liftA routes to pair.A ref")
		testkit.Equal(t, sut.B.n, 0, "liftA does not touch pair.B")
	})
}

func TestLiftB(t *testing.T) {
	t.Parallel()
	inner := incActionB("Count")
	lifted := liftB[*counterA, *counterB](inner)

	testkit.Equal(t, lifted.Name, "B.Count", "liftB prefixes with B.")
	testkit.Equal(t, lifted.Kind, FailureInvariant, "liftB preserves Kind")

	rapid.Check(t, func(rt *rapid.T) {
		sut := pair[*counterA, *counterB]{A: &counterA{}, B: &counterB{}}
		ref := pair[*counterA, *counterB]{A: &counterA{}, B: &counterB{}}
		lifted.Run(rt, sut, ref)
		testkit.Equal(t, sut.B.n, 1, "liftB routes to pair.B SUT")
		testkit.Equal(t, ref.B.n, 1, "liftB routes to pair.B ref")
		testkit.Equal(t, sut.A.n, 0, "liftB does not touch pair.A")
	})
}

func TestLiftTripleA(t *testing.T) {
	t.Parallel()
	inner := incAction("Op")
	lifted := liftTripleA[*counterA, *counterB, *counterC](inner)

	testkit.Equal(t, lifted.Name, "A.Op", "liftTripleA prefixes with A.")

	rapid.Check(t, func(rt *rapid.T) {
		sut := triple[*counterA, *counterB, *counterC]{A: &counterA{}, B: &counterB{}, C: &counterC{}}
		ref := triple[*counterA, *counterB, *counterC]{A: &counterA{}, B: &counterB{}, C: &counterC{}}
		lifted.Run(rt, sut, ref)
		testkit.Equal(t, sut.A.n, 1, "routes to triple.A")
		testkit.Equal(t, sut.B.n, 0, "does not touch triple.B")
		testkit.Equal(t, sut.C.n, 0, "does not touch triple.C")
	})
}

func TestLiftTripleB(t *testing.T) {
	t.Parallel()
	inner := incActionB("Op")
	lifted := liftTripleB[*counterA, *counterB, *counterC](inner)

	testkit.Equal(t, lifted.Name, "B.Op", "liftTripleB prefixes with B.")

	rapid.Check(t, func(rt *rapid.T) {
		sut := triple[*counterA, *counterB, *counterC]{A: &counterA{}, B: &counterB{}, C: &counterC{}}
		ref := triple[*counterA, *counterB, *counterC]{A: &counterA{}, B: &counterB{}, C: &counterC{}}
		lifted.Run(rt, sut, ref)
		testkit.Equal(t, sut.B.n, 1, "routes to triple.B")
		testkit.Equal(t, sut.A.n, 0, "does not touch triple.A")
		testkit.Equal(t, sut.C.n, 0, "does not touch triple.C")
	})
}

func TestLiftTripleC(t *testing.T) {
	t.Parallel()
	inner := incActionC("Op")
	lifted := liftTripleC[*counterA, *counterB, *counterC](inner)

	testkit.Equal(t, lifted.Name, "C.Op", "liftTripleC prefixes with C.")

	rapid.Check(t, func(rt *rapid.T) {
		sut := triple[*counterA, *counterB, *counterC]{A: &counterA{}, B: &counterB{}, C: &counterC{}}
		ref := triple[*counterA, *counterB, *counterC]{A: &counterA{}, B: &counterB{}, C: &counterC{}}
		lifted.Run(rt, sut, ref)
		testkit.Equal(t, sut.C.n, 1, "routes to triple.C")
		testkit.Equal(t, sut.A.n, 0, "does not touch triple.A")
		testkit.Equal(t, sut.B.n, 0, "does not touch triple.B")
	})
}
