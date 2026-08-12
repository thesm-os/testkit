// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
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
	// the subject, resolved against the interface. Ptr addresses the literal,
	// for a stateful law whose Check lives on the pointer.
	Ctor *sdk.Expr
	Args []sdk.Ref
	Ptr  bool

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

	// Const is a constant field's qualified value — a sentinel the
	// declaration stamped, rendered where a manifest names its stamp key.
	Const *sdk.Expr
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
	// Selection composes per method, but a claim holds over the interface —
	// the sticky stamp rides the reader and negates the writer-earned
	// observability law — so the conflict scan runs against every method's
	// mixins, partners included: an excluded method's claim still holds.
	claims := map[string]bool{}
	for i := range harness.Methods {
		for _, name := range harness.Methods[i].Mixins {
			claims[name] = true
		}
	}
	for i := range harness.Methods {
		m := &harness.Methods[i]
		if _, partner := partners[m.Name]; partner {
			continue
		}
		for _, r := range tiers.Select(classificationsOf(m), paramsOf(m)) {
			if reason, negated := negatedBy(claims, r.Law); negated {
				b.Unbound = append(b.Unbound, Skip{Method: r.Law, Reason: reason})
				continue
			}
			if binding, ok := lawOf(b, harness, r, m, keyed); ok {
				b.Laws = append(b.Laws, binding)
			}
		}
	}
}

// negatedBy resolves the first conflict row a held claim triggers, in the
// table's own order so the generated header is deterministic.
func negatedBy(claims map[string]bool, law string) (string, bool) {
	for _, n := range tiers.LawNegations() {
		if n.Law == law && claims[n.Mixin] {
			return n.Reason, true
		}
	}
	return "", false
}

// lawOf fills one rule, false where [Bindings.Unbound] records why not.
func lawOf(b *Bindings, harness *suite.Contract, r tiers.Rule, m, keyed *suite.Method) (*LawBinding, bool) {
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
		Ptr:      spec.Ptr,
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
		field, reason := lawFieldOf(b, harness, f, m, keyed)
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
func lawFieldOf(b *Bindings, harness *suite.Contract, f tiers.Field, m, keyed *suite.Method) (*LawField, string) {
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
	case tiers.KindTrace:
		// The runner binds the trace on any law implementing TraceBinder;
		// a generated value would race the binding it already gets.
		return nil, ""
	case tiers.KindSupplied:
		if f.Optional {
			// The manifest says zero is sound: the law reads the field's
			// absence as the claim's unrefined form, so the binding omits it
			// and the option that would fill it stays a consumer's choice.
			return nil, ""
		}
		return nil, f.Name + " waits on the " + f.From + " option, which no generated value can stand in for"
	case tiers.KindRole:
		role, reason := roleMethod(b, harness, f.From, m, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		if len(role.CallArgs()) > 1 {
			// The role templates compose a single-input closure; a composite
			// call would render with the wrong arity and fail in whichever
			// package armed it.
			return nil, f.Name + " closes over " + role.Name +
				", which takes several inputs no single-value closure composes"
		}
		if (f.Name == "Drain" || f.Name == "Collect") && !returnsSlice(role) {
			// The drain spelling returns the slice the method returns; an
			// iterator-shaped method needs a collect loop this build does not
			// compose.
			return nil, f.Name + " drains " + role.Name +
				", which streams through an iterator rather than returning a slice"
		}
		field.Method = role.Name
		field.TakesCtx = role.TakesContext()
		if f.Name == "Drain" || f.Name == "Collect" {
			// The drained element type, not the pool's: a collector's slice
			// element is what the law compares, and the two agree by the
			// reference derivation's own check.
			if elem, err := collectorElem(b, role); err == "" {
				field.Value = elem
			}
		}
		return field, ""
	case tiers.KindConstant:
		value, ok := stampValue(m, f.From)
		if !ok {
			return nil, f.Name + " reads the " + f.From + " stamp, which this declaration does not carry"
		}
		pkg, name, qualified := splitQualified(value)
		if !qualified {
			return nil, f.Name + "'s stamp names " + value + ", which carries no package to import it from"
		}
		field.Const = sdk.NewExternal(pkg, name)
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
	}
	return nil, f.Name + " has the unknown kind " + string(f.Kind)
}

// roleMethod resolves a manifest role to the method whose call fills it:
// the selecting method itself, a shape family, or a partner the selecting
// method's own stamp names.
func roleMethod(b *Bindings, harness *suite.Contract, from string, m, keyed *suite.Method) (*suite.Method, string) {
	switch from {
	case "self":
		return m, ""
	case "family.reader":
		if keyed == nil {
			return nil, "names the reader family, and the interface has no keyed reader"
		}
		return keyed, ""
	}
	if mixin, param, ok := strings.Cut(from, "."); ok && !strings.HasPrefix(from, "family.") {
		v, stamped := shape.MixinParamKey(mixin, param).Get(m.Source.Meta())
		if !stamped || v == "" {
			return nil, "names " + from + ", which the selecting method does not stamp"
		}
		role := methodOf(harness, golang.LocalName(v))
		if role == nil {
			return nil, "names " + from + " = " + v + ", which is not a method of " + b.IfaceName
		}
		return role, ""
	}
	return nil, "names " + from + ", which nothing resolves"
}

// returnsSlice reports whether the method's first result is a slice.
func returnsSlice(m *suite.Method) bool {
	return len(m.Returns) > 0 && m.Returns[0].Source != nil &&
		shape.GoSliceElem(m.Returns[0].Source) != nil
}

// stampValue reads one classification parameter off the selecting method, by
// the raw key the manifest spells — the annotator's own composition, reached
// through the registry rather than respelled.
func stampValue(m *suite.Method, key string) (string, bool) {
	v, ok := sdk.EnsureKey(key, sdk.StringParser).Get(m.Source.Meta())
	return v, ok && v != ""
}

// splitQualified splits a resolver-qualified name into its package path and
// trailing identifier.
func splitQualified(v string) (pkg, name string, ok bool) {
	i := strings.LastIndexByte(v, '.')
	if i <= 0 || i == len(v)-1 {
		return "", "", false
	}
	return v[:i], v[i+1:], true
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
