// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// unstamped is a projection method whose source carries no classification
// parameters at all — the smallest thing a stamp read can miss on.
func unstamped() *subject.Method {
	return &subject.Method{Sig: &golang.Sig{Source: &node.Method{}}}
}

// ─── The arm walk ────────────────────────────────────────────────
//
// Every refusal arm below is a header line in a generated file: the tests
// hold each to a reason that names what is missing, because a silent nil
// field is a law that runs, passes, and asserts nothing.

func namedRef(name string) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefNamed, Name: name}
}

func pkgRef(pkg, name string) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefNamed, Package: pkg, Name: name}
}

func sliceRef(elem *node.TypeRef) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefSlice, Elem: elem}
}

func ctxRef() *node.TypeRef {
	r := pkgRef("context", "Context")
	golang.MetaIsContext.Set(r.EnsureMeta(), true, "test")
	return r
}

func chanRef() *node.TypeRef {
	r := namedRef("chan")
	golang.MetaIsChannel.Set(r.EnsureMeta(), true, "test")
	return r
}

// projectedReturns is [projected] with the raw returns handed through — for
// a return whose Source carries stamps the res helper cannot spell.
func projectedReturns(name string, params []golang.Param, returns []golang.Return) *subject.Method {
	src := &node.Method{Name: name}
	return &subject.Method{Sig: &golang.Sig{
		Name: name, Params: params, Returns: returns, Source: src,
	}}
}

// projected builds a projection method by hand: the signature the arms read,
// with the classification stamps the tests choose.
func projected(name string, params []golang.Param, returns []golang.Return) *subject.Method {
	src := &node.Method{Name: name}
	return &subject.Method{Sig: &golang.Sig{
		Name: name, Params: params, Returns: returns, Source: src,
	}}
}

func arg(name string, src *node.TypeRef) golang.Param {
	return golang.Param{Name: name, Type: sdk.Builtin(src.Name), Source: src}
}

func res(src *node.TypeRef) golang.Return {
	return golang.Return{Type: sdk.Builtin(src.Name), Source: src, Error: golang.IsError(src)}
}

func stamp(m *subject.Method, shapeName, keyQ, valueQ string) *subject.Method {
	bag := m.Source.EnsureMeta()
	if shapeName != "" {
		shape.MetaShape.Set(bag, shapeName, "test")
	}
	if keyQ != "" {
		shape.MetaKeyType.Set(bag, keyQ, "test")
	}
	if valueQ != "" {
		shape.MetaValueType.Set(bag, valueQ, "test")
	}
	return m
}

// roleRule wraps one role field into the smallest rule that carries it.
func roleRule(law, field string) tiers.Rule {
	return tiers.Rule{Law: law, Fields: []tiers.Field{
		{Name: field, Kind: tiers.KindRole, From: "self"},
	}}
}

// bindField runs the role dispatch for one law/field/method triple.
func bindField(b *Bindings, law, field string, m *subject.Method) (*LawField, string) {
	r := roleRule(law, field)
	return lawFieldOf(b, nil, r, r.Fields[0], m, nil)
}

// The spellings these tests repeat.
const (
	famWriter = "family.writer"
	qStr      = "string"
)

// harnessOf wraps methods into the projection lawsOf walks.
func harnessOf(methods ...*subject.Method) *subject.Projection {
	h := &subject.Projection{}
	for _, m := range methods {
		h.Methods = append(h.Methods, *m)
	}
	return h
}

// funcRef builds a func-kinded type ref — the callable a compute-taking or
// body-taking role declares.
func funcRef() *node.TypeRef {
	return &node.TypeRef{Name: "func", TypeKind: node.TypeRefFunc}
}
