// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/suite"
)

// TestSaturationDerivation pins the prover's own derivation: the wardrobe's
// kinds per method shape, the session laws' reachable pair, the unwearable
// skip, and the one law whose kill criterion is the differential itself.
func TestSaturationDerivation(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))

	t.Run("the kinds follow the method's shape", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{
			Subject: suite.Subject{IfaceName: "Mixed"},
			Values:  Pool{Type: sdk.Builtin("Value"), Q: "Value", Field: "V", OtherField: "VOther"},
		}
		kinds := func(m *suite.Method) []string {
			out := make([]string, 0, 4)
			for _, sm := range satMutantsOf(b, m) {
				out = append(out, sm.Kind)
			}
			return out
		}

		reader := stamp(projected("Get",
			[]golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
			[]golang.Return{res(namedRef("Value")), errRet}), "", "", "Value")
		testkit.Equal(t, kinds(reader),
			[]string{"inert", "flicker", "sputter", "spill", "flap"},
			"a pool-typed reader spills and flaps beside the shared kinds")

		scalar := projected("Count", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("int")), errRet})
		testkit.Equal(t, kinds(scalar),
			[]string{"inert", "flicker", "sputter", "spill", "wane", "wax"},
			"an integer scalar wanes and waxes")

		replay := projected("Replay", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(sliceRef(namedRef("Entry"))), errRet})
		// fade alternates; the other three do not. A claim read off a single
		// drain is answered by whichever call the law makes, and the prover
		// isolates one law at a time — so an every-second defect lands on the
		// parity the law never sees.
		testkit.Equal(t, kinds(replay),
			[]string{"inert", "flicker", "sputter", "fade", "jumble", "dupdrain", "flood"},
			"a slice reader fades, jumbles, repeats and floods")

		page := projected("Page",
			[]golang.Param{arg("ctx", ctxRef()), arg("cur", namedRef("Cursor"))},
			[]golang.Return{
				res(sliceRef(namedRef("Value"))), res(namedRef("Cursor")), res(namedRef("bool")), errRet,
			})
		testkit.Equal(t, kinds(page), []string{"inert", "flicker", "sputter", "echo"},
			"a page-shaped walk echoes")

		// An operation reporting only an error carries the pair no
		// alternating defect can express: one that refuses after its first
		// call, and one that silently drops after it. An idempotence law
		// calls twice and discards the first answer, so a defect on every
		// other call is absorbed by the call nobody reads.
		op := projected("Close", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{errRet})
		testkit.Equal(t, kinds(op),
			[]string{"inert", "flicker", "sputter", "stick", "latch"},
			"an error-only operation sticks and latches")

		// A method taking a computation promises how often it runs it, and
		// only invoking that computation can break the promise.
		compute := projected("Run",
			[]golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr)), arg("fn", funcRef())},
			[]golang.Return{res(namedRef(qStr)), errRet})
		testkit.Assert(t, kinds(compute)).Contains("greedy",
			"a nullary computation parameter earns the extra invocation")

		// One that takes arguments does not: calling it would mean inventing
		// its inputs here, and a defect supplying its own inputs is testing
		// something else.
		fed := funcRef()
		fed.FuncParams = []*node.TypeRef{namedRef(qStr)}
		callback := projected("Each",
			[]golang.Param{arg("ctx", ctxRef()), arg("fn", fed)},
			[]golang.Return{errRet})
		for _, k := range kinds(callback) {
			testkit.NotEqual(t, k, "greedy", "a callback taking arguments is left alone")
		}

		// Every shape above carries flicker, and that is the point: a claim
		// about two calls agreeing is broken by an answer that changes, and
		// no shape-specific kind supplies one. Only a method answering
		// nothing has no flicker to wear.
		void := projected("Ping", []golang.Param{arg("ctx", ctxRef())}, nil)
		testkit.Equal(t, kinds(void), []string{"inert"},
			"a method answering nothing has nothing to flicker")
	})

	t.Run("a streamed result answers empty rather than nil", func(t *testing.T) {
		t.Parallel()
		// A stream's zero value is a nil function, and ranging over one
		// panics — so a wear answering the zero takes the run down before
		// the law it was worn for is asked anything.
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		seq2 := projected("List", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(pkgRef("iter", "Seq2"))})
		for _, sm := range satMutantsOf(b, seq2) {
			testkit.Equal(t, sm.Seq, 2, "the arity rides every wear on the method")
			testkit.Equal(t, sm.SeqHelper(), "EmptySeq2", "and names its helper")
		}

		seq1 := projected("Each", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(pkgRef("iter", "Seq"))})
		testkit.Equal(t, satMutantsOf(b, seq1)[0].SeqHelper(), "EmptySeq",
			"the one-value form names its own")

		// Each stream defect names the runtime helper that wears it, at the
		// arity the signature declares. A wrong name here is a template that
		// renders a call to a function that does not exist, which the corpus
		// catches — but only after a regeneration, and only for the arities
		// the corpus happens to hold.
		defects := map[string]string{}
		for _, sm := range satMutantsOf(b, seq2) {
			defects[sm.Kind] = sm.SeqDefect()
		}
		testkit.Equal(t, defects[kindFadeSeq], "FadedSeq2", "the faded drain")
		testkit.Equal(t, defects[kindDupSeq], "DoubledSeq2", "the doubled one")
		testkit.Equal(t, defects[kindFlood], "FloodedSeq2", "and the one that will not end")
		testkit.Equal(t, satMutantsOf(b, seq1)[0].SeqDefect(), "FadedSeq",
			"the one-value arity spells its own")

		plain := projected("Get", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("Value")), errRet})
		testkit.Equal(t, satMutantsOf(b, plain)[0].SeqHelper(), "",
			"a result that is not a stream names no helper")

		// The near misses, because the check is a name match against the
		// standard library and a wrong one would dress a stream defect on a
		// method that cannot carry it.
		notIter := projected("Chan", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(pkgRef("example.com/x", "Seq2"))})
		testkit.Equal(t, seqArity(notIter), 0, "Seq2 from another package is not one")
		wrongName := projected("Iter", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(pkgRef("iter", "Pull"))})
		testkit.Equal(t, seqArity(wrongName), 0, "iter has more than the two sequence types")
		twoResults := projected("Both", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(pkgRef("iter", "Seq")), errRet})
		testkit.Equal(t, seqArity(twoResults), 0,
			"a sequence beside an error is not the shape the drains take")
	})

	t.Run("the boundary wear needs a bound to cross", func(t *testing.T) {
		t.Parallel()
		// Built from the law's own stamped constant, so a law without one —
		// or with one no integer can be read from — has no line to step over
		// and earns no wear rather than a wear that steps nowhere.
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		count := projected("Count", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("int")), errRet})
		h := harnessOf(count)

		noBound := &LawBinding{
			ID:     lawid.CountEqualsReference,
			Fields: []*LawField{{Name: "Count", Method: "Count"}},
		}
		_, ok := overshootOf(b, h, noBound, "Count")
		testkit.False(t, ok, "no stamped bound, no boundary to cross")

		fractional := &LawBinding{ID: lawid.AggregatorBounded, Fields: []*LawField{
			{Name: "Count", Method: "Count"}, {Name: fieldMax, Lit: "1.5"},
		}}
		_, ok = overshootOf(b, h, fractional, "Count")
		testkit.False(t, ok, "no counting shape answers one past a fraction")

		unread := &LawBinding{
			ID:     lawid.AggregatorBounded,
			Fields: []*LawField{{Name: fieldMax, Lit: "5"}},
		}
		_, ok = overshootOf(b, h, unread, "Count")
		testkit.False(t, ok, "a bound on a method the law does not read is not this law's line")
	})

	t.Run("the surface knows its reach and its restatement", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		b.Session = &SessionSpec{Reader: "Get", Writer: "Store"}
		b.Laws = []*LawBinding{
			{ID: lawid.MonotonicReads, Session: true},
			{ID: lawid.CountEqualsReference, Fields: []*LawField{{Name: "Count", Method: "Count"}}},
			{ID: lawid.PoolBalanced},
		}
		get := projected("Get", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
			[]golang.Return{res(namedRef("Value")), errRet})
		store := projected("Store", []golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))},
			[]golang.Return{errRet})
		count := projected("Count", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("int")), errRet})
		saturationOf(b, harnessOf(get, store, count))

		testkit.Equal(t, b.SatLaws[0].Methods, []string{"Get", "Store"},
			"a trace law wears defects on the session pair")
		testkit.True(t, b.SatLaws[1].AcceptSemantic,
			"the differential restated accepts the differential's own divergence")
		testkit.True(t, b.SatLaws[2].Unwearable,
			"a door-only law is skipped by name, never doomed")
	})
}

// A witnessed interface emits no prover: its wrappers would need the
// witness instantiation the surface does not thread.
func TestSaturationSkipsWitnessedInterfaces(t *testing.T) {
	t.Parallel()

	b := &Bindings{
		Subject:   suite.Subject{IfaceName: "Store"},
		Witnesses: []sdk.Ref{sdk.Builtin(qStr)},
		Laws:      []*LawBinding{{ID: lawid.ReadAfterWrite}},
	}
	saturationOf(b, harnessOf())
	testkit.Len(t, b.SatLaws, 0, "no prover over a generic surface")
}

// A law naming a method the projection does not carry wears nothing there —
// the prover's wardrobe stays honest about what it can dress.
func TestSaturationSkipsUnprojectedMethods(t *testing.T) {
	t.Parallel()

	b := &Bindings{
		Subject: suite.Subject{IfaceName: "Mixed"},
		Laws: []*LawBinding{{
			ID:     lawid.ReadAfterWrite,
			Fields: []*LawField{{Name: "Read", Method: "Nonesuch"}},
		}},
	}
	saturationOf(b, harnessOf())
	testkit.Equal(t, b.SatLaws[0].Methods, []string{"Nonesuch"},
		"the law still names its reach")
	testkit.Len(t, b.SatMutants, 0, "and nothing unprojected is dressed")
}
