// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/suite"
)

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
