// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/testkit"
	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/suite"
	"go.thesmos.sh/testkit/generator/suite/projection"
)

// deriveCtx is the context parameter every ctx-taking fixture method
// shares; [golang.IsContext] answers from the qualified spelling.
func deriveCtx() golang.Param {
	return golang.Param{Name: "ctx", Source: storefixture.PkgNamed("context", "Context")}
}

// storeGet is the fully-armed shape: context, one draw, a named
// result beside the error — every signature family reaches it.
func storeGet() suite.Method {
	return suite.Method{
		Sig: &golang.Sig{
			Name:   "Get",
			Params: []golang.Param{deriveCtx(), keyParam("key")},
			Returns: []golang.Return{
				{Source: storefixture.Named("Value")},
				{Error: true},
			},
		},
		ArgFields: []string{"Key"},
	}
}

// storeIface pairs the methods with a fixture that can deliver every
// draw the fixtures above declare.
func storeIface(ctxDeclared bool, methods ...suite.Method) suite.Iface {
	return suite.Iface{
		Name: "Store", Token: "store", CtxDeclared: ctxDeclared, Methods: methods,
		Fixture: suite.Fixture{Fields: []suite.FixtureField{{
			Name:   "Key",
			Sample: golang.Sample{Text: `"k"`},
			Other:  golang.Sample{Text: `"o"`},
		}}},
	}
}

// familyCase is one method shape and the ID set its rules license.
type familyCase struct {
	name  string
	iface suite.Iface
	want  []vocab.ID
}

func (c familyCase) Name() string { return c.name }

func TestSignatureDerivesTheFamilies(t *testing.T) {
	t.Parallel()

	closeM := suite.Method{Sig: &golang.Sig{
		Name:    "Close",
		Params:  []golang.Param{deriveCtx()},
		Returns: []golang.Return{{Error: true}},
	}}
	noCtx := suite.Method{Sig: &golang.Sig{
		Name:    "Len",
		Returns: []golang.Return{{Source: storefixture.Named("int")}, {Error: true}},
	}}
	noErr := suite.Method{
		Sig: &golang.Sig{
			Name:    "Peek",
			Params:  []golang.Param{deriveCtx(), keyParam("key")},
			Returns: []golang.Return{{Source: storefixture.Named("Value")}},
		},
		ArgFields: []string{"Key"},
	}

	testkit.TableTest(t, []familyCase{
		{
			"a fully-armed method reaches every family",
			storeIface(true, storeGet()),
			[]vocab.ID{"Get/smoke", "Get/cancel", "Get/nilcontext", "Get/deadline", "Get/zero-on-error"},
		},
		{
			"a teardown-shaped method never carries deadline",
			storeIface(true, closeM),
			[]vocab.ID{"Close/smoke", "Close/cancel", "Close/nilcontext"},
		},
		{
			"without the ctx directive only the smoke derives",
			storeIface(false, storeGet()),
			[]vocab.ID{"Get/smoke"},
		},
		{
			"a context-less method carries no context families",
			storeIface(true, noCtx),
			[]vocab.ID{"Len/smoke"},
		},
		{
			"no error result, no zero family",
			storeIface(true, noErr),
			[]vocab.ID{"Peek/smoke", "Peek/cancel", "Peek/nilcontext", "Peek/deadline"},
		},
		{
			"declared totality excludes the zero family alone",
			storeIface(true, func() suite.Method {
				m := storeGet()
				m.Mixins = []string{suite.MixinTotal}
				return m
			}()),
			[]vocab.ID{"Get/smoke", "Get/cancel", "Get/nilcontext", "Get/deadline"},
		},
	}, func(t *testing.T, tc familyCase) {
		plans, refusals := suite.Signature{}.Derive(tc.iface)
		testkit.Len(t, refusals, 0, "derivable shapes refuse nothing")
		got := make([]vocab.ID, len(plans))
		for i, p := range plans {
			got[i] = p.ID.Render()
		}
		testkit.Equal(t, got, tc.want, "the rules license exactly these checks, in family order")
	})
}

func TestSignatureShapesTheChecks(t *testing.T) {
	t.Parallel()

	plans, _ := suite.Signature{}.Derive(storeIface(true, storeGet()))
	byID := map[vocab.ID]projection.CheckPlan{}
	for _, p := range plans {
		byID[p.ID.Render()] = p
	}
	wantCall := projection.CallPlan{
		Method: "Get",
		Args:   []projection.Expr{projection.ExprCtx, projection.FixtureCall("store", "Key")},
	}

	t.Run("the smoke survives with the fixture draw", func(t *testing.T) {
		t.Parallel()
		p := byID["Get/smoke"]
		testkit.Equal(t, p.Body, projection.Body(projection.SmokeSurvives{Call: wantCall}),
			"the smoke body carries the derived call")
		testkit.Equal(t, p.Defect, projection.Defect(projection.StubPanic{Option: projection.OptionName("Store", "Get")}),
			"the smoke is proven by the panicking double")
		testkit.Equal(t, p.Class, vocab.ClassSmoke, "class buckets the report")
	})

	t.Run("nilcontext is proven by the accepting double", func(t *testing.T) {
		t.Parallel()
		p := byID["Get/nilcontext"]
		testkit.Equal(t, p.Defect, projection.Defect(projection.AcceptsNil{Option: projection.OptionName("Store", "Get")}),
			"the claim's stronger arm — returns an error — needs the accepting defect")
	})

	t.Run("the context families share the call and the swap defect", func(t *testing.T) {
		t.Parallel()
		for _, id := range []vocab.ID{"Get/cancel", "Get/deadline"} {
			p := byID[id]
			testkit.Equal(t, p.Defect, projection.Defect(projection.CtxSwap{Option: projection.OptionName("Store", "Get")}),
				"a context family is proven by the context-ignoring double")
		}
	})

	t.Run("every plan is Proven", func(t *testing.T) {
		t.Parallel()
		for id, p := range byID {
			testkit.Equal(t, p.Falsifiable.State, vocab.FalsifiableProven,
				"the signature families all carry planted defects: "+string(id))
		}
	})
}

func TestSignatureRefusesUnderivableDraws(t *testing.T) {
	t.Parallel()

	entry := suite.Method{
		Sig: &golang.Sig{
			Name:    "Append",
			Params:  []golang.Param{deriveCtx(), {Name: "e", Source: storefixture.Named("Entry")}},
			Returns: []golang.Return{{Error: true}},
		},
		ArgFields: []string{"Entry"},
	}
	iface := suite.Iface{Name: "Log", Token: "log", CtxDeclared: true, Methods: []suite.Method{entry}}

	plans, refusals := suite.Signature{}.Derive(iface)
	testkit.Len(t, plans, 0, "an underivable draw silences no single family — it refuses them all")
	testkit.Len(t, refusals, 1, "the whole family set folds into one refusal")
	testkit.Equal(t, refusals[0].What, "Append's signature checks", "the refusal names the method's family set")
	testkit.Contains(t, refusals[0].Why, "Entry", "the refusal names the draw nothing supplies")
	testkit.Contains(t, refusals[0].Remedy, "LogWithFixture", "the remedy names the consumer seam")
}

func TestSignaturePlansSatisfyTheInventory(t *testing.T) {
	t.Parallel()

	plans, _ := suite.Signature{}.Derive(storeIface(true, storeGet()))
	inv := projection.Inventory{Iface: "Store", Token: "store", Checks: plans}
	testkit.NoError(t, inv.Verify(), "derived plans hold the inventory's parity rules by construction")
}
