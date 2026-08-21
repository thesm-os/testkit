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

func TestIdentityCompared(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	fn := &node.TypeRef{TypeKind: node.TypeRefFunc}
	ptr := &node.TypeRef{TypeKind: node.TypeRefPointer, Elem: namedRef("Value")}

	testkit.False(t, identityCompared(projected("Total", nil,
		[]golang.Return{res(namedRef("int")), errRet})), "a value compares by value")
	testkit.True(t, identityCompared(projected("Watch", nil,
		[]golang.Return{res(chanRef()), errRet})), "a channel compares by identity")
	testkit.True(t, identityCompared(projected("Hook", nil,
		[]golang.Return{{Type: sdk.Builtin("func"), Source: fn}, errRet})),
		"a function compares by identity")
	testkit.True(t, identityCompared(projected("Find", nil,
		[]golang.Return{{Type: sdk.Builtin("ptr"), Source: ptr}, errRet})),
		"a pointer compares by identity")
	testkit.False(t, identityCompared(projected("Fire", nil, nil)),
		"nothing returned, nothing compared")
}

func TestContractParamNames(t *testing.T) {
	t.Parallel()

	testkit.True(t, len(contractParamNames("codec")) > 0,
		"a registered contract lists its parameters")
	testkit.True(t, contractParamNames("nonesuch") == nil,
		"an unregistered one lists nothing")
}

// TestAnsweringWriterDetection pins the shape the write-ordering laws hold
// out for: one value in, the same type out beside the error.
func TestAnsweringWriterDetection(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	valueRef := pkgRef("example.com/s", "Value")

	up := projected("Persist",
		[]golang.Param{arg("ctx", ctxRef()), arg("v", valueRef)},
		[]golang.Return{res(pkgRef("example.com/s", "Value")), errRet})
	plain := projected("Put",
		[]golang.Param{arg("ctx", ctxRef()), arg("v", valueRef)},
		[]golang.Return{errRet})
	crossed := projected("Save",
		[]golang.Param{arg("ctx", ctxRef()), arg("v", valueRef)},
		[]golang.Return{res(namedRef("int64")), errRet})

	h := &suite.Contract{Methods: []suite.Method{*plain, *up, *crossed}}
	found := answeringWriterOf(h)
	testkit.True(t, found != nil && found.Name == "Persist",
		"the answered-state write is the answering writer")

	none := &suite.Contract{Methods: []suite.Method{*plain, *crossed}}
	testkit.True(t, answeringWriterOf(none) == nil,
		"an error-only write and a scalar-answering write both hide the stored state")
}

// TestSubscribeShape pins the subscription closure: the handle is kept for
// the drain, never compared, and the shape takes nothing.
func TestSubscribeShape(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{Subject: suite.Subject{IfaceName: "Contract"}}
	field, reason := bindField(b, lawid.PublisherDelivers, "Subscribe",
		projected("Subscribe", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(pkgRef("example.com/p", "Handle")), errRet}))
	testkit.True(t, reason == "" && field.Out != nil, "a nullary subscription binds: "+reason)

	_, reason = bindField(b, lawid.PublisherDelivers, "Subscribe",
		projected("Subscribe", []golang.Param{arg("ctx", ctxRef()), arg("topic", namedRef(qStr))},
			[]golang.Return{res(pkgRef("example.com/p", "Handle")), errRet}))
	testkit.Assert(t, reason).Contains("no subscription draw supplies", "a topic is an input nothing draws")
}
