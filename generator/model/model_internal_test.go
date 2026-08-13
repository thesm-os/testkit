// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/suite"
)

// TestTemplateSurfaces pins the accessors the templates read: each answers a
// constant the render pass composes imports from, and a wrong one is a
// generated file that fails to compile in whichever package arms it.
func TestTemplateSurfaces(t *testing.T) {
	t.Parallel()

	b := &Bindings{}
	testkit.Equal(t, b.ModelPkg(), ModelPkg, "the runner's import path")
	testkit.Equal(t, b.LinearizePkg(), LinearizePkg, "the Porcupine wiring's import path")
	testkit.Equal(t, (&Action{}).ModelPkg(), ModelPkg, "and the actions' own view of it")
}

// TestNeedsFixture pins the fixture obligation: the property constructs the
// derived inputs exactly when something reads them — a pool, a law anchored
// on a fixture key, a per-position pair — because an unused local is a
// compile error in a generated file.
func TestNeedsFixture(t *testing.T) {
	t.Parallel()

	testkit.False(t, (&Bindings{}).NeedsFixture(), "nothing read, nothing built")
	testkit.True(t, (&Bindings{LawsUseFixture: true}).NeedsFixture(),
		"a fixture-anchored law obliges it")
	testkit.True(t, (&Bindings{Actions: []*Action{{Pool: poolKeys}}}).NeedsFixture(),
		"a drawing pool obliges it")
	testkit.True(t, (&Bindings{Actions: []*Action{{Args: []ActionArg{{Field: "A"}}}}}).NeedsFixture(),
		"a per-position pair obliges it")
}

// TestStampedSentinel pins the declared-sentinel resolution: a contract error
// row naming a stamped parameter hands the oracle the declaration's own error
// identity, and every other shape of the stamp falls back to minting.
func TestStampedSentinel(t *testing.T) {
	t.Parallel()

	carrier := &suite.Method{Sig: &golang.Sig{Source: &node.Method{}}}
	carrier.Contracts = []string{"lease"}

	t.Run("an unnamed parameter mints", func(t *testing.T) {
		t.Parallel()
		_, stamped := stampedSentinel(nil, carrier, "lease", "")
		testkit.False(t, stamped, "no parameter, no stamp to read")
	})

	t.Run("an unstamped parameter mints", func(t *testing.T) {
		t.Parallel()
		_, stamped := stampedSentinel(nil, carrier, "lease", "held")
		testkit.False(t, stamped, "the declaration said nothing")
	})

	t.Run("an unqualified stamp mints", func(t *testing.T) {
		t.Parallel()
		bare := &suite.Method{Sig: &golang.Sig{Source: &node.Method{}}}
		bare.Contracts = []string{"lease"}
		shape.ContractParamKey("lease", "held").Set(bare.Source.EnsureMeta(), "ErrHeld", "test")
		_, stamped := stampedSentinel(nil, bare, "lease", "held")
		testkit.False(t, stamped, "a bare name carries no package to import")
	})

	t.Run("a qualified stamp is the oracle's sentinel", func(t *testing.T) {
		t.Parallel()
		host := &suite.Method{Sig: &golang.Sig{Source: &node.Method{}}}
		host.Contracts = []string{"lease"}
		shape.ContractParamKey("lease", "held").
			Set(host.Source.EnsureMeta(), "example.com/lease.ErrHeld", "test")
		sym, stamped := stampedSentinel(nil, host, "lease", "held")
		testkit.True(t, stamped && sym != nil, "one identity for the oracle and the law")
	})
}

// TestWitnessSpelling pins the stamp vocabulary a witness answers in: bare
// for a builtin, package-qualified for a source type, empty for a form no
// stamp ever spells.
func TestWitnessSpelling(t *testing.T) {
	t.Parallel()

	testkit.Equal(t, witnessSpelling(sdk.Builtin("int")), "int", "a builtin is its own spelling")
	testkit.Equal(t, witnessSpelling(sdk.External("example.com/x", "Score")),
		"example.com/x.Score", "a named type carries its package")
	testkit.Equal(t, witnessSpelling(nil), "", "and nothing else spells")
}

// TestTemplateImportAccessors pins the paths the templates qualify
// through: each is a constant the emit layer registers as an import.
func TestTemplateImportAccessors(t *testing.T) {
	t.Parallel()
	b := &Bindings{}
	testkit.True(t, b.TracePath() != "" && b.LawPath() != "",
		"the trace and law packages are spelled for the classifier and the doors")
}

// TestSubstQ pins the substitution's two arms: a bound parameter name lands
// at its witness, everything else passes through.
func TestSubstQ(t *testing.T) {
	t.Parallel()

	b := &Bindings{witnessQ: map[string]string{"V": "int"}}
	testkit.Equal(t, b.substQ("V"), "int", "a parameter name lands at its witness")
	testkit.Equal(t, b.substQ("string"), "string", "a concrete spelling passes through")
}

// txTrio builds a tx-stamped harness: a begin carrying the contract with the
// given partner keys, plus whatever siblings the case declares.
func txTrio(t *testing.T, beginReturns []golang.Return, siblings ...*suite.Method) *suite.Contract {
	t.Helper()
	begin := projected("Begin", []golang.Param{arg("ctx", ctxRef())}, beginReturns)
	begin.Contracts = []string{"tx"}
	shape.ContractRoleKey("tx").Set(begin.Source.EnsureMeta(), "begin", "test")
	methods := make([]suite.Method, 0, 1+len(siblings))
	methods = append(methods, *begin)
	for _, s := range siblings {
		methods = append(methods, *s)
	}
	return &suite.Contract{Methods: methods}
}

// TestContractActionRepointGuards walks the arms that keep the composite
// honest: half a trio, a begin answering no handle, and a role driving
// nothing leave the sequences exactly as the shapes chose them.
func TestContractActionRepointGuards(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	handleRet := res(pkgRef("example.com/t", "Tx"))
	// The annotator stamps each resolved partner with its role key, which
	// is what the role map reads back.
	terminal := func(name, role string) *suite.Method {
		m := projected(name,
			[]golang.Param{arg("ctx", ctxRef()), arg("h", pkgRef("example.com/t", "Tx"))},
			[]golang.Return{errRet})
		shape.ContractRoleKey("tx").Set(m.Source.EnsureMeta(), role, "test")
		return m
	}
	beginAction := func() *Action { return &Action{Method: "Begin", Shape: shapeAggregator} }

	t.Run("half a trio leaves the shapes' choice", func(t *testing.T) {
		t.Parallel()
		h := txTrio(t, []golang.Return{handleRet, errRet}, terminal("Commit", "commit"))
		b := &Bindings{Actions: []*Action{beginAction(), {Method: "Commit", Shape: shapeWriter}}}
		contractActionsOf(b, h)
		testkit.Equal(t, b.Actions[0].Shape, shapeAggregator,
			"a trio missing its rollback composes nothing")
	})

	t.Run("a begin answering no handle is left alone", func(t *testing.T) {
		t.Parallel()
		h := txTrio(t, []golang.Return{errRet},
			terminal("Commit", "commit"), terminal("Rollback", "rollback"))
		b := &Bindings{Actions: []*Action{
			beginAction(),
			{Method: "Commit", Shape: shapeWriter},
			{Method: "Rollback", Shape: shapeWriter},
		}}
		contractActionsOf(b, h)
		testkit.Equal(t, b.Actions[0].Shape, shapeAggregator,
			"there is no handle to thread, so there is no composite")
	})

	t.Run("a terminal driving nothing blocks the composite", func(t *testing.T) {
		t.Parallel()
		h := txTrio(t, []golang.Return{handleRet, errRet},
			terminal("Commit", "commit"), terminal("Rollback", "rollback"))
		b := &Bindings{Actions: []*Action{beginAction(), {Method: "Commit", Shape: shapeWriter}}}
		contractActionsOf(b, h)
		testkit.Equal(t, b.Actions[0].Shape, shapeAggregator,
			"the composite consumes both terminals or composes nothing")
	})

	t.Run("a role method driving nothing is skipped", func(t *testing.T) {
		t.Parallel()
		h := txTrio(t, []golang.Return{handleRet, errRet},
			terminal("Commit", "commit"), terminal("Rollback", "rollback"))
		b := &Bindings{Actions: []*Action{{Method: "Commit", Shape: shapeWriter}}}
		contractActionsOf(b, h)
		testkit.Equal(t, b.Actions[0].Shape, shapeWriter,
			"no begin action, no composite to hang the terminals on")
	})

	t.Run("the whole trio composes and consumes", func(t *testing.T) {
		t.Parallel()
		h := txTrio(t, []golang.Return{handleRet, errRet},
			terminal("Commit", "commit"), terminal("Rollback", "rollback"))
		b := &Bindings{Actions: []*Action{
			beginAction(),
			{Method: "Commit", Shape: shapeWriter},
			{Method: "Rollback", Shape: shapeWriter},
		}}
		contractActionsOf(b, h)
		testkit.Equal(t, len(b.Actions), 1, "the terminals are consumed")
		testkit.Equal(t, b.Actions[0].Shape, "tx.begin", "and the begin drives the cycle")
		testkit.Equal(t, b.Actions[0].TxCommit, "Commit", "threading its own handle forward")
		testkit.Equal(t, len(b.Skipped), 2, "with the header saying where the terminals went")
	})
}

// TestAppendLegGuards holds the append leg to its own eligibility: an
// appender whose offsets are not int64, or whose method drives nothing,
// derives no leg.
func TestAppendLegGuards(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	run := func(offset *node.TypeRef) *suite.Method {
		m := projected("Run",
			[]golang.Param{arg("ctx", ctxRef()), arg("v", pkgRef("example.com/a", "Value"))},
			[]golang.Return{res(offset), errRet})
		m.Contracts = []string{"appender"}
		return m
	}

	t.Run("a non-int64 offset keeps the sequential law alone", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Actions: []*Action{{Method: "Run", Pool: poolValues}}}
		a, _ := appendActionOf(b, &suite.Contract{Methods: []suite.Method{*run(namedRef("string"))}})
		testkit.True(t, a == nil, "the shared-history model counts in int64")
	})

	t.Run("an undriven appender derives no leg", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{}
		a, _ := appendActionOf(b, &suite.Contract{Methods: []suite.Method{*run(namedRef("int64"))}})
		testkit.True(t, a == nil, "no action, nothing to interleave")
	})

	t.Run("a driven int64 appender derives the leg", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Actions: []*Action{{Method: "Run", Pool: poolValues}}}
		concurrentOf(b, &suite.Contract{Methods: []suite.Method{*run(namedRef("int64"))}}, nil, nil)
		testkit.Equal(t, b.ConcFamily, concFamilyAppend, "the offsets join one shared history")
		testkit.True(t, b.ConcEntry != nil, "typed at the method's own entry")
	})
}

// TestCASLegGuards holds the cell leg to a whole pair: a VersionedCell
// without its aggregator read interleaves nothing worth checking.
func TestCASLegGuards(t *testing.T) {
	t.Parallel()

	b := &Bindings{
		Reference: Reference{Oracle: OracleContract, ContractStore: "VersionedCell", VersionField: "V"},
		Actions:   []*Action{{Method: "Swap", Shape: shapeCASWriter, Pool: poolKeys}},
	}
	concurrentOf(b, &suite.Contract{}, nil, nil)
	testkit.Equal(t, b.ConcFamily, "", "half a pair drives nothing Porcupine can order")

	b.Actions = append(b.Actions, &Action{Method: "Get", Shape: shapeAggregator})
	concurrentOf(b, &suite.Contract{}, nil, nil)
	testkit.Equal(t, b.ConcFamily, concFamilyCAS, "the whole pair derives the cell leg")

	testkit.Equal(t, (&Bindings{}).CasMismatch(), CtorErr{},
		"no error rows, no identity for the model to match")
}
