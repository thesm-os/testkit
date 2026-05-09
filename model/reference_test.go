// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/model"
)

// --- Minimal types for lift tests ---

type (
	counterA struct{ n int }
	counterB struct{ n int }
	counterC struct{ n int }
)

func incAction(name string) model.Action[*counterA] {
	return model.Action[*counterA]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(_ *rapid.T, sut, ref *counterA) model.ActionResult {
			sut.n++
			ref.n++
			return model.ActionResult{}
		},
	}
}

func incActionB(name string) model.Action[*counterB] {
	return model.Action[*counterB]{
		Name: name,
		Kind: model.FailureInvariant,
		Run: func(_ *rapid.T, sut, ref *counterB) model.ActionResult {
			sut.n++
			ref.n++
			return model.ActionResult{}
		},
	}
}

func incActionC(name string) model.Action[*counterC] {
	return model.Action[*counterC]{
		Name: name,
		Run: func(_ *rapid.T, sut, ref *counterC) model.ActionResult {
			sut.n++
			ref.n++
			return model.ActionResult{}
		},
	}
}

func TestLiftA(t *testing.T) {
	t.Parallel()
	inner := incAction("Inc")
	lifted := model.LiftA[*counterA, *counterB](inner)

	testkit.Equal(t, lifted.Name, "A.Inc", "LiftA prefixes with A.")
	testkit.Equal(t, lifted.Kind, model.FailureSemantic, "LiftA preserves Kind")

	rapid.Check(t, func(rt *rapid.T) {
		sut := model.Pair[*counterA, *counterB]{A: &counterA{}, B: &counterB{}}
		ref := model.Pair[*counterA, *counterB]{A: &counterA{}, B: &counterB{}}
		lifted.Run(rt, sut, ref)
		testkit.Equal(t, sut.A.n, 1, "LiftA routes to pair.A SUT")
		testkit.Equal(t, ref.A.n, 1, "LiftA routes to pair.A ref")
		testkit.Equal(t, sut.B.n, 0, "LiftA does not touch pair.B")
	})
}

func TestLiftB(t *testing.T) {
	t.Parallel()
	inner := incActionB("Count")
	lifted := model.LiftB[*counterA, *counterB](inner)

	testkit.Equal(t, lifted.Name, "B.Count", "LiftB prefixes with B.")
	testkit.Equal(t, lifted.Kind, model.FailureInvariant, "LiftB preserves Kind")

	rapid.Check(t, func(rt *rapid.T) {
		sut := model.Pair[*counterA, *counterB]{A: &counterA{}, B: &counterB{}}
		ref := model.Pair[*counterA, *counterB]{A: &counterA{}, B: &counterB{}}
		lifted.Run(rt, sut, ref)
		testkit.Equal(t, sut.B.n, 1, "LiftB routes to pair.B SUT")
		testkit.Equal(t, ref.B.n, 1, "LiftB routes to pair.B ref")
		testkit.Equal(t, sut.A.n, 0, "LiftB does not touch pair.A")
	})
}

func TestLiftTripleA(t *testing.T) {
	t.Parallel()
	inner := incAction("Op")
	lifted := model.LiftTripleA[*counterA, *counterB, *counterC](inner)

	testkit.Equal(t, lifted.Name, "A.Op", "LiftTripleA prefixes with A.")

	rapid.Check(t, func(rt *rapid.T) {
		sut := model.Triple[*counterA, *counterB, *counterC]{A: &counterA{}, B: &counterB{}, C: &counterC{}}
		ref := model.Triple[*counterA, *counterB, *counterC]{A: &counterA{}, B: &counterB{}, C: &counterC{}}
		lifted.Run(rt, sut, ref)
		testkit.Equal(t, sut.A.n, 1, "routes to triple.A")
		testkit.Equal(t, sut.B.n, 0, "does not touch triple.B")
		testkit.Equal(t, sut.C.n, 0, "does not touch triple.C")
	})
}

func TestLiftTripleB(t *testing.T) {
	t.Parallel()
	inner := incActionB("Op")
	lifted := model.LiftTripleB[*counterA, *counterB, *counterC](inner)

	testkit.Equal(t, lifted.Name, "B.Op", "LiftTripleB prefixes with B.")

	rapid.Check(t, func(rt *rapid.T) {
		sut := model.Triple[*counterA, *counterB, *counterC]{A: &counterA{}, B: &counterB{}, C: &counterC{}}
		ref := model.Triple[*counterA, *counterB, *counterC]{A: &counterA{}, B: &counterB{}, C: &counterC{}}
		lifted.Run(rt, sut, ref)
		testkit.Equal(t, sut.B.n, 1, "routes to triple.B")
		testkit.Equal(t, sut.A.n, 0, "does not touch triple.A")
		testkit.Equal(t, sut.C.n, 0, "does not touch triple.C")
	})
}

func TestLiftTripleC(t *testing.T) {
	t.Parallel()
	inner := incActionC("Op")
	lifted := model.LiftTripleC[*counterA, *counterB, *counterC](inner)

	testkit.Equal(t, lifted.Name, "C.Op", "LiftTripleC prefixes with C.")

	rapid.Check(t, func(rt *rapid.T) {
		sut := model.Triple[*counterA, *counterB, *counterC]{A: &counterA{}, B: &counterB{}, C: &counterC{}}
		ref := model.Triple[*counterA, *counterB, *counterC]{A: &counterA{}, B: &counterB{}, C: &counterC{}}
		lifted.Run(rt, sut, ref)
		testkit.Equal(t, sut.C.n, 1, "routes to triple.C")
		testkit.Equal(t, sut.A.n, 0, "does not touch triple.A")
		testkit.Equal(t, sut.B.n, 0, "does not touch triple.B")
	})
}
