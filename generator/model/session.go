// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"slices"
	"strings"

	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/suite"
	"go.thesmos.sh/testkit/generator/tiers"
)

// sessionSpecOf derives the per-client classification the session laws
// share — memoized on the bindings, because the classifier is one file-level
// function however many laws read through it.
//
// The write-ordering laws need the version the store assigned to each write,
// and a writer answering only an error hides it: the trace records what was
// sent, never what was stamped. Those laws bind only beside an
// upserter-shaped write — (ctx, V) (V, error), the stored state answered —
// and refuse otherwise with the shape that would carry them.
func sessionSpecOf(
	b *Bindings, harness *suite.Contract, r tiers.Rule, m, keyed *suite.Method,
) (*SessionSpec, string) {
	if keyed == nil {
		return nil, "orders reads no keyed reader here answers"
	}
	mixin := ""
	if len(r.Needs) > 0 {
		mixin = r.Needs[0]
	}
	version, given := shape.MixinParamKey(mixin, "version").Get(m.Source.Meta())
	if !given || version == "" {
		return nil, "names no version= member, and a session guarantee is defined against the value's ordering stamp"
	}
	if r.Law != lawid.MonotonicReads && answeringWriterOf(harness) == nil {
		return nil, "orders writes the trace cannot see — the writer answers only an error, " +
			"and the version the store assigned dies with the call; an answering " +
			"write (ctx, V) (V, error) surfaces it"
	}
	if b.Session != nil {
		return b.Session, ""
	}
	if b.sessionKeyField == "" {
		return nil, "keys per client on a value member no convention names"
	}
	value, _, why := resultType(keyed)
	if why != "" {
		return nil, why
	}
	if b.Keys.Type == nil {
		return nil, "instantiates at a key type no method here draws"
	}
	spec := &SessionSpec{
		ClassifyName: strings.ToLower(b.IfaceName[:1]) + b.IfaceName[1:] + "SessionClassify",
		Reader:       keyed.Name,
		Value:        value,
		KeyField:     b.sessionKeyField,
		VersionField: version,
		Key:          b.Keys.Type,
	}
	if up := answeringWriterOf(harness); up != nil {
		spec.Writer = up.Name
	}
	b.Session = spec
	return spec, ""
}

// SessionSpec is the derived per-client classification: the one closure the
// session laws share, spelled once at file level so the sequential property
// and the concurrent leg reference the same derivation.
type SessionSpec struct {
	// ClassifyName is the generated file-level function's identifier.
	ClassifyName string

	// Reader is the keyed read the classifier interprets, with TakesCtx
	// mirrored for the header.
	Reader string

	// Value is the read's result type; KeyField and VersionField are its
	// identity and ordering members.
	Value                  sdk.Ref
	KeyField, VersionField string

	// Writer is the upserter-shaped write whose answered state carries the
	// stamp — empty where no write surfaces one, which classifies writes
	// out and binds the read-ordering law alone.
	Writer string

	// Key is the pool key type the laws instantiate at.
	Key sdk.Ref
}

// sessionVersionOf reports the first session mixin carrying a version=
// param anywhere in the method set: the carrying method, the member it
// names, and whether one was found.
func sessionVersionOf(harness *suite.Contract) (carrier *suite.Method, member string, stamped bool) {
	for i := range harness.Methods {
		m := &harness.Methods[i]
		for _, mx := range m.Mixins {
			if !slices.Contains(sessionMixins, mx) {
				continue
			}
			if v, given := shape.MixinParamKey(mx, "version").Get(m.Source.Meta()); given && v != "" {
				return m, v, true
			}
		}
	}
	return nil, "", false
}

// versionFieldDiag holds version= to the value struct's own fields. Every
// projection of the ordering stamp is a field selector — the session
// classifier reads it, and the cas cell assigns it (v.Rev = cur.Rev + 1),
// which no method form can satisfy — so a method or a missing member is
// refused here by name. Without the refusal the stamp passes every layer
// unvalidated and the failure surfaces as a build error in the consumer's
// package, attributed to generated code rather than to the directive that
// caused it. A value type whose struct declaration is out of reach passes
// through: the compile keeps that case honest, and refusing what cannot be
// seen would break a witnessed value spelled by its parameter name.
func versionFieldDiag(ctx *sdk.GeneratorContext, iface *sdk.Interface, valueQ, member string) bool {
	var s *sdk.Struct
	for cand := range ctx.Reader.Structs().All() {
		if cand.Package+"."+cand.Name == valueQ {
			s = cand
			break
		}
	}
	if s == nil {
		return true
	}
	for _, f := range s.Fields {
		if f.Name == member {
			return true
		}
	}
	for _, m := range s.Methods {
		if m != nil && m.Name == member {
			ctx.Diag.Errorf(iface.Pos(),
				"%s: version=%q on %q names a method of %s; the ordering stamp is "+
					"read and assigned as a field, and no method can stand there",
				Name, member, iface.Name, valueQ)
			return false
		}
	}
	ctx.Diag.Errorf(iface.Pos(),
		"%s: version=%q on %q names no member of %s",
		Name, member, iface.Name, valueQ)
	return false
}
