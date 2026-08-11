// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/suite"
	"go.thesmos.sh/testkit/generator/tiers"
)

// LawFieldKindPrefix composes each law field's emit kind —
// `model.lawfield.<Name>` — which is the template that renders it.
//
// Dispatch is on the field's name because the catalogue already asserts the
// name is the concept: Values is always the value pool, Read is always the
// observation the claim compares against. A field unique to one law gets its
// own template under the same rule.
const LawFieldKindPrefix = "model.lawfield."

// LawBinding is one law, instantiated and filled, in the generated registry.
type LawBinding struct {
	sdk.BaseEmit

	// ID is the identifier the law reports under, for the header.
	ID string

	// Ctor is the law struct, qualified; Args are its type arguments after
	// the subject, resolved against the interface.
	Ctor *sdk.Expr
	Args []sdk.Ref

	// Fields fill the struct, each through its name's template.
	Fields []*LawField
}

// Kind returns the one template every binding renders through.
func (*LawBinding) Kind() sdk.Kind { return "model.law" }

// LawField is one filled field of a law struct.
type LawField struct {
	sdk.BaseEmit

	// KindName selects the field's template by its name.
	KindName sdk.Kind

	// Name is the struct field, for the composite literal.
	Name string

	// Method is the role method a closure field calls, with TakesCtx saying
	// whether the call forwards the run's context.
	Method   string
	TakesCtx bool

	// Iface, Key and Value spell the closure's parameter and result types.
	Iface, Key, Value sdk.Ref

	// Pool names the shared local a generator field reuses, and KeyOfName the
	// shared key projection a handle field reuses — the same values the
	// actions and the derived reference already draw from, which is the
	// one-derivation rule inside the file.
	Pool, KeyOfName string
}

// Kind returns the field's template key.
func (f *LawField) Kind() sdk.Kind { return f.KindName }

// ModelPkg surfaces the runner's import path to the field templates, whose
// closures take the runner's *T.
func (*LawField) ModelPkg() string { return ModelPkg }

// lawsOf selects and fills every law the interface's classifications earn.
//
// Selection is [tiers.Select] over each non-partner method's whole
// classification set. A selected rule that cannot be filled lands in
// [Bindings.Unbound] with what it is waiting on — rendered in the header,
// because a law that quietly failed to bind reads as a claim the run checks.
func lawsOf(b *Bindings, harness *suite.Contract, partners map[string]string, keyed *suite.Method) {
	for i := range harness.Methods {
		m := &harness.Methods[i]
		if _, partner := partners[m.Name]; partner {
			continue
		}
		for _, r := range tiers.Select(classificationsOf(m), paramsOf(m)) {
			if binding, ok := lawOf(b, r, m, keyed); ok {
				b.Laws = append(b.Laws, binding)
			}
		}
	}
}

// lawOf fills one rule, false where [Bindings.Unbound] records why not.
func lawOf(b *Bindings, r tiers.Rule, m, keyed *suite.Method) (*LawBinding, bool) {
	spec, specified := tiers.BindingFor(r.Law)
	if !specified {
		b.Unbound = append(b.Unbound, Skip{
			Method: r.Law,
			Reason: "the catalogue carries no instantiation spec for it",
		})
		return nil, false
	}

	lb := &LawBinding{
		BaseEmit: b.BaseEmit,
		ID:       r.Law,
		Ctor:     sdk.NewExternal(LawPkg, spec.Type),
		// The subject leads every law's argument list; the spec spells only
		// what follows it.
		Args: []sdk.Ref{b.IfaceRef},
	}
	for _, a := range spec.Args {
		switch a {
		case tiers.BindKey:
			lb.Args = append(lb.Args, b.Keys.Type)
		case tiers.BindValue:
			lb.Args = append(lb.Args, b.Values.Type)
		}
	}

	for _, f := range r.Fields {
		field, reason := lawFieldOf(b, f, m, keyed)
		if reason != "" {
			b.Unbound = append(b.Unbound, Skip{Method: r.Law, Reason: reason})
			return nil, false
		}
		if field != nil {
			lb.Fields = append(lb.Fields, field)
		}
	}
	return lb, true
}

// lawFieldOf fills one manifest entry: a field, nil for one the law defaults,
// or the reason nothing can fill it.
func lawFieldOf(b *Bindings, f tiers.Field, m, keyed *suite.Method) (*LawField, string) {
	field := &LawField{
		BaseEmit: b.BaseEmit,
		KindName: sdk.Kind(LawFieldKindPrefix + f.Name),
		Name:     f.Name,
		Iface:    b.IfaceRef,
		Key:      b.Keys.Type,
		Value:    b.Values.Type,
	}

	switch f.Kind {
	case tiers.KindDefault:
		// The law's Check defaults it; a generated value would be a second
		// opinion about a number the law already owns.
		return nil, ""
	case tiers.KindRole:
		role, reason := roleMethod(f.From, m, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		field.Method = role.Name
		field.TakesCtx = role.TakesContext()
		return field, ""
	case tiers.KindGenerator:
		field.Pool = f.From
		return field, ""
	case tiers.KindHandle:
		if b.Reference.KeyField == "" {
			return nil, f.Name + " needs the key projection, which was not derivable here"
		}
		field.KeyOfName = b.KeyOfName()
		return field, ""
	case tiers.KindConstant, tiers.KindTrace, tiers.KindSupplied:
		return nil, f.Name + " is a " + string(f.Kind) + " field, which nothing renders"
	}
	return nil, f.Name + " has the unknown kind " + string(f.Kind)
}

// roleMethod resolves a manifest role to the method whose call fills it.
func roleMethod(from string, m, keyed *suite.Method) (*suite.Method, string) {
	switch from {
	case "self":
		return m, ""
	case "family.reader":
		if keyed == nil {
			return nil, "names the reader family, and the interface has no keyed reader"
		}
		return keyed, ""
	}
	return nil, "names " + from + ", which nothing resolves"
}

// classificationsOf is the method's whole set, in one namespace: its detector
// shape, its mixins, and the contracts it fills a role in.
func classificationsOf(m *suite.Method) []string {
	out := []string{}
	if s := shape.Get(m.Source.Meta()); s != "" {
		out = append(out, s)
	}
	out = append(out, m.Mixins...)
	return append(out, m.Contracts...)
}

// paramsOf collects the classification parameters the When clauses condition
// on, keyed the way [tiers.Condition.Param] spells them.
func paramsOf(m *suite.Method) map[string]string {
	out := map[string]string{}
	for _, name := range m.Mixins {
		for _, p := range mixinParamNames(name) {
			if v, ok := shape.MixinParamKey(name, p).Get(m.Source.Meta()); ok {
				out[shape.MixinParamKey(name, p).Name()] = v
			}
		}
	}
	return out
}

// mixinParamNames returns the named mixin's declared parameters — the
// registry's fact, like the sibling scan in model.go.
func mixinParamNames(name string) []string {
	for _, mx := range mixins.All() {
		if mx.Name == name {
			return mx.Params
		}
	}
	return nil
}
