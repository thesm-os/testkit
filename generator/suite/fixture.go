// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/compositewriter"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/multiargwriter"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/writer"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/builder"
)

// OtherSuffix names the companion field holding a second, different value for
// the same parameter.
//
// Two values rather than one, for every parameter: a check comparing a result
// against a single input passes whenever the subject happened to be seeded with
// it, and a miss check whose key happens to hit asserts nothing and reports
// success. The pair is what makes both able to fail.
const OtherSuffix = "Other"

// FixtureField is one derived input, with the second value that makes a check
// able to fail.
type FixtureField struct {
	// Name is the exported field the generated struct declares — the Pascal
	// form of the parameter's identifier.
	Name string

	// Type is the parameter's type, rendered through the backend so the file
	// registers whatever import it needs.
	Type sdk.Ref

	// Sample and Other are the two derived values, for a parameter whose type
	// yields one whole. Empty for a struct, whose value is composed from Parts.
	Sample, Other golang.Sample

	// Parts is the per-field pair for a struct parameter, in declaration order.
	//
	// A struct's value is composed rather than carried as text, because a field
	// whose own type is a struct needs its type spelled beside its braces — and
	// only the backend knows how to spell it for this file, and to register the
	// import it needs. Text alone renders `{F: "x"}`, which is not a value.
	Parts []FixturePart

	// Variadic reports that the parameter this field was derived from was
	// declared `...T`, so the field holds one element rather than the list the
	// method takes.
	//
	// Carried only to be said out loud in the generated file. Nothing about the
	// derivation changes — [golang.Param] keeps Type as the element type, which
	// is the type of the one value a check is handed.
	Variadic bool

	// Companion calls the type's `<Type>Defaults()`, and wins over Sample where
	// the source declares one.
	//
	// Only the sample half. Other is "a value that should not be found", which
	// is what a miss check needs and is a different claim from "a value this
	// type accepts" — one function cannot answer both, and asking for a second
	// convention to supply the alternate would cost more than the miss check
	// gains.
	Companion *sdk.Expr
}

// FixturePart is one field of a composed struct value.
type FixturePart struct {
	// Name is the field's identifier in the composite literal.
	Name string

	// Sample and Other are the two values for it.
	Sample, Other golang.Sample
}

// Composed reports whether this field's value is built from Parts rather than
// carried whole.
func (f FixtureField) Composed() bool { return len(f.Parts) > 0 }

// FixtureValue is one of a field's two values, flattened for rendering.
//
// The template needs "this field's sample" and "this field's alternate" spelled
// identically, and text/template cannot pass which one it wants down to a
// sub-template. Choosing here keeps one spelling instead of two loops that
// could drift.
type FixtureValue struct {
	Type  sdk.Ref
	Value golang.Sample
	Parts []FixtureValuePart
}

// FixtureValuePart is one field of a composed [FixtureValue].
type FixtureValuePart struct {
	Name  string
	Value golang.Sample
}

// Choose flattens this field to one of its two values.
func (f FixtureField) Choose(alternate bool) FixtureValue {
	out := FixtureValue{Type: f.Type, Value: f.Sample}
	if alternate {
		out.Value = f.Other
	}
	for _, p := range f.Parts {
		v := p.Sample
		if alternate {
			v = p.Other
		}
		out.Parts = append(out.Parts, FixtureValuePart{Name: p.Name, Value: v})
	}
	return out
}

// OtherName is the identifier of the companion field.
func (f FixtureField) OtherName() string { return f.Name + OtherSuffix }

// OK reports whether a value for this field could be produced at all — a
// companion, or both halves of a derived pair.
//
// A parameter whose type admits no literal and declares no companion — a
// channel, a func, a type from a package the run never read — yields neither,
// and the one check whose meaning is the value is dropped rather than emitted
// against something nobody could write.
func (f FixtureField) OK() bool {
	return f.Companion != nil || f.Composed() || (f.Sample.OK() && f.Other.OK())
}

// Reason phrases why nothing could be derived for this field.
//
// Only [golang.RefusedNoLiteral] is a fact about the type. The rest describe
// this run's own input — a package the patterns did not reach, a walk that hit
// its budget — and reporting one of those as settled sends an author to change
// source that is already correct.
func (f FixtureField) Reason() string {
	if f.Sample.Refusal.Incomplete() {
		return "which this run did not resolve, so no value was derived for it"
	}
	return "which no literal can be written for"
}

// Fixture is the derived input set for one interface.
type Fixture struct {
	// TypeName is the generated struct's identifier — `<Iface>Fixture`.
	TypeName string

	// CtorName is the identifier of the function returning the derived values,
	// which a consumer reads to see what they would be overriding.
	CtorName string

	Fields []FixtureField

	// groups records which parameter each field was derived from, so a check
	// can name the field its own argument landed in — which is not the
	// parameter's name wherever two types contest one.
	groups []paramGroup
}

// FieldFor names the fixture field a method's parameter is supplied from.
//
// Falls back to the parameter's own name, which is what the fixture calls a
// field no other method contests. Every caller asks about a parameter of a
// method the fixture was built from, so the fallback answers the same thing the
// loop would — and a `""` would compose `cfg.Fixture.` into generated source
// rather than failing where a reader could see it.
func (f Fixture) FieldFor(p golang.Param) string {
	for _, g := range f.groups {
		if g.param.Field == p.Field && g.param.Source.Equal(p.Source) {
			return g.name
		}
	}
	return p.Field
}

// Field returns the field of that name, and whether one was derived.
func (f Fixture) Field(name string) (FixtureField, bool) {
	for _, x := range f.Fields {
		if x.Name == name {
			return x, true
		}
	}
	return FixtureField{}, false
}

// fixtureOf derives one input per distinct parameter across the method set.
//
// Which parameters share a field, and how a name two types contest is spelled,
// is [groupParams]. What is decided here is what each group is filled with: the
// type's `<Type>Defaults()` where the source declares one, the composed parts of
// a struct, and the derived pair otherwise.
func fixtureOf(ctx *sdk.GeneratorContext, iface *sdk.Interface, methods []Method) Fixture {
	f := Fixture{
		TypeName: iface.Name + "Fixture",
		CtorName: "Default" + iface.Name + "Fixture",
	}
	f.groups = groupParams(methods)
	for _, g := range f.groups {
		sample, other := sampleFor(g.param, ctx.Reader)
		f.Fields = append(f.Fields, FixtureField{
			Name:      g.name,
			Type:      g.param.Type,
			Variadic:  g.param.Variadic,
			Sample:    sample,
			Other:     other,
			Parts:     partsFor(g.param, ctx.Reader),
			Companion: companionFor(ctx, g.param.Source),
		})
	}
	return f
}

// paramGroup is one fixture field: a parameter name at one type, and the first
// method that introduced it.
type paramGroup struct {
	// name is the field's identifier, which is the parameter's own where no
	// other method takes that name at a different type.
	name string

	// method is the first method in method-set order to take this pair, which
	// is what disambiguates a name two types share.
	method string

	param golang.Param
}

// groupParams collects the interface's parameters into one field per name and
// type, in method-set order.
//
// # Why the pair rather than the name
//
// A `key string` on the reader and one on the deleter are the same value as far
// as a conformance run is concerned, and giving them separate fields would let a
// consumer override one and silently not the other.
//
// But a name is not a type. `Put(ctx, s Session)` beside `Get(ctx, s string)` is
// ordinary Go — nothing stops two methods naming their parameters alike — and a
// fixture keyed on the name alone holds one of them and hands it to the method
// that takes the other, which does not compile. An earlier version diagnosed
// that and told the author to rename a parameter, which is bad advice about
// correct source.
//
// # How a shared name is disambiguated
//
// By the method that introduced each type, not by the type itself: a composite
// has no name to spell, and `SSlice` would be a spelling this package invented.
// `PutS` and `GetS` name something the reader can find in the source. Only a
// contested name is qualified; a name carrying one type keeps it.
//
// The qualified spelling can in principle meet an uncontested parameter
// literally named `PutS`. Nothing here detects that, and nothing needs to: two
// fields of one name is a struct the toolchain refuses, so the cost is a
// compile error over generated source rather than a check quietly handed the
// wrong value.
func groupParams(methods []Method) []paramGroup {
	var groups []paramGroup
	byField := map[string]int{}
	for _, m := range methods {
		for _, p := range m.CallArgs() {
			if findGroup(groups, p) {
				continue
			}
			groups = append(groups, paramGroup{name: p.Field, method: m.Name, param: p})
			byField[p.Field]++
		}
	}
	// Qualify every group whose parameter name another type also claims, so
	// neither spelling is privileged by the order the walk happened to take.
	for i := range groups {
		if byField[groups[i].param.Field] > 1 {
			groups[i].name = groups[i].method + groups[i].param.Field
		}
	}
	return groups
}

// findGroup reports whether a group already holds this parameter's name and
// type.
func findGroup(groups []paramGroup, p golang.Param) bool {
	for _, g := range groups {
		if g.param.Field == p.Field && g.param.Source.Equal(p.Source) {
			return true
		}
	}
	return false
}

// withChecks selects each method's family and drops what the fixture cannot
// supply.
//
// One pass rather than two, and after the fixture exists: a check names the
// field it is handed, and which field that is depends on whether another method
// contests the parameter's name.
//
// A check handed a value that cannot be written down is a check that does not
// compile, and one handed a zero where a real value was meant passes against a
// subject that does nothing. Dropping is the honest answer, and the generated
// file names each absence with the parameter that caused it.
func withChecks(
	c *sdk.Provenance, ctx *sdk.GeneratorContext, iface *sdk.Interface, f Fixture, methods []Method,
) []Method {
	out := make([]Method, 0, len(methods))
	for _, m := range methods {
		kept := make([]*Check, 0, 5)
		for _, ck := range signatureChecks(c, iface, f, m) {
			if missing, field, ok := undeliverable(f, ck); ok {
				ctx.Diag.Warnf(iface.Pos(),
					"%s: %s.%s takes %s, %s, so its "+
						"%q check is not generated; supply one through %sWithFixture "+
						"and write the check as %sOn%s",
					Name, iface.Name, m.Name, missing, field.Reason(), ck.Subtest,
					iface.Name, iface.Name, m.Name)
				continue
			}
			kept = append(kept, ck)
		}
		// No emptiness guard: smoke is emitted for every method and needs no
		// derived input, so a method always keeps at least one check.
		m.Checks = kept
		m.ArgFields = fixtureArgs(f, m, false)
		out = append(out, m)
	}
	return out
}

// undeliverable names the first argument the fixture cannot supply, for a check
// that needs one derived.
//
// A check that does not is never dropped: every type has a zero value, the
// fixture declares a field for every parameter, and a field nothing could be
// derived for is left at that zero rather than omitted. So a method taking a
// callback still gets its context family, and only the check whose semantics
// depend on the value goes.
func undeliverable(f Fixture, c *Check) (string, FixtureField, bool) {
	if !c.NeedsDerivedInput {
		return "", FixtureField{}, false
	}
	// The alternate is a companion of its sample, so both are derivable or
	// neither is; looking the base name up answers for the pair.
	return undeliverableArgs(f, c.Args)
}

// sampleFor derives one parameter's pair of values.
//
// For a struct it sets *every* exported field it can, which is where this
// departs from [golang.SampleRefFor] — and deliberately. That function sets the
// first settable field and says why: a sample exists to be distinguishable, and
// one field achieves that while staying readable. Right for a builder, where
// the value is handed to a setter and read straight back.
//
// A conformance suite asks a different question. An implementation that
// silently drops a field passes every check built from a sample that never set
// it, and two values differing in one field are indistinguishable to a subject
// keyed on another. Discrimination is the whole point here, and readability is
// the thing to trade for it.
//
// The per-field values are still eidos's — only the policy of how many to set
// is testkit's, and that is a testing decision rather than a fact about Go.
// Nesting is left to [golang.SampleRefFor], so a struct inside a struct gets
// eidos's one-field form: the field that matters is the parameter's own.
func sampleFor(p golang.Param, r golang.Resolver) (sample, alternate golang.Sample) {
	return golang.SampleRefFor(p.Source, p.Name, r)
}

// partsFor composes a struct parameter's value field by field.
//
// Every exported field it can, which is where this departs from
// [golang.SampleRefFor] — and deliberately. That function sets the first
// settable field and says why: a sample exists to be distinguishable, and one
// field achieves that while staying readable. Right for a builder, where the
// value is handed to a setter and read straight back.
//
// A conformance suite asks a different question. An implementation that
// silently drops a field passes every check built from a sample that never set
// it, and two values differing in one field are indistinguishable to a subject
// keyed on another. Discrimination is the point here, and readability is the
// thing to trade for it.
//
// The per-field values are still eidos's — only the policy of how many to set
// is testkit's, and that is a testing decision rather than a fact about Go.
// Each is carried as a [golang.Sample] rather than as text, so a field whose own
// type is a struct keeps the reference the backend needs to spell it.
func partsFor(p golang.Param, r golang.Resolver) []FixturePart {
	decl, resolved := r.Resolve(p.Source)
	s, ok := decl.(*sdk.Struct)
	if !resolved || !ok {
		return nil
	}

	var parts []FixturePart
	for _, f := range golang.ExportedFields(s) {
		inner, innerAlt := golang.SampleRefFor(f.Type, f.Name, r)
		if !inner.OK() {
			// A field no literal can be written for is left at its zero rather
			// than losing the whole sample: the fields around it still
			// discriminate, and refusing here would drop every check the
			// parameter feeds.
			continue
		}
		parts = append(parts, FixturePart{Name: f.Name, Sample: inner, Other: innerAlt})
	}
	return parts
}

// companionFor returns a call to the type's `<Type>Defaults()` function, or nil
// when the source declares none.
//
// The one place a team already writes down "here is a valid instance of this
// type". A derived sample is plausible strings, and a type with real validation
// — an identifier that must be a UUID, an address that must hold an `@` —
// accepts only some of them. Deriving one anyway means the first run of a
// correct implementation is a wall of failures that are the fixture's fault,
// which is how a suite gets switched off.
//
// Found by looking rather than by being told, which is [builder.CompanionSuffix]'s
// own rule — the function is an ordinary declaration and the convention is
// shared, so a package that wrote one for its builder gets this for free.
//
// The signature is checked rather than only the name: a `PayloadDefaults`
// taking arguments, or returning something else, is a different function that
// happens to collide, and calling it would emit a fixture that does not compile.
func companionFor(ctx *sdk.GeneratorContext, t *sdk.TypeRef) *sdk.Expr {
	if t == nil || t.Name == "" {
		return nil
	}
	name := t.Name + builder.CompanionSuffix
	fn, found := ctx.Reader.Functions().Where(func(fn *sdk.Function) bool {
		return fn.Name == name && fn.Package == t.Package
	}).First()
	if !found || len(fn.Params) != 0 || len(fn.Returns) != 1 {
		return nil
	}
	if r := fn.Returns[0].Type; r == nil || r.Name != t.Name {
		return nil
	}
	return sdk.NewExternal(t.Package, name)
}

// seedOf names the writer a harness populates its subject through, or nil when
// the interface declares none.
//
// A reader over an empty subject asserts nothing, so something has to write
// first — and for any interface carrying a writer, that something is the
// interface itself. Nothing is asked of the consumer: the shape annotator has
// already said which method writes, and the fixture already holds a value of
// what it takes.
//
// A read-only interface over external state has no writer, and is the case a
// consumer's own seed exists for.
//
// The first writer in method-set order rather than a choice between several.
// Two writers over one value type is a shape this cannot resolve — and where
// they differ, an author who cares supplies a seed rather than being asked
// which is meant.
func seedOf(f Fixture, methods []Method) *Seed {
	for _, m := range methods {
		if !writesSomething(m) || !m.ReturnsError() {
			continue
		}
		args := fixtureArgs(f, m, false)
		if _, _, undeliverable := undeliverableArgs(f, args); undeliverable {
			continue
		}
		return &Seed{Method: m, Args: args}
	}
	return nil
}

// writesSomething reports whether the shape annotator classified this method as
// a write.
//
// Three detectors rather than one. They differ only in arity — `writer` takes a
// single non-context argument, `compositewriter` two, `multiargwriter` three or
// more — and the seed passes whatever the method declares, so arity is not
// something it has to know. Keying on `writer` alone left a `Put(ctx, key, v)`
// interface unable to seed itself, which is the ordinary keyed store.
//
// `mutator` is deliberately absent even though it writes: it returns nothing, so
// a seed through one cannot report its own failure, and a seed that fails
// silently leaves every check after it asserting against an empty subject. That
// exclusion is [Seed]'s error return restated, and [seedOf]'s ReturnsError guard
// would refuse it anyway.
func writesSomething(m Method) bool {
	switch shape.Get(m.Source.Meta()) {
	case writer.Name, compositewriter.Name, multiargwriter.Name:
		return true
	default:
		return false
	}
}

// Seed is the write a harness populates each fresh subject with.
type Seed struct {
	// Method is the writer the seed calls.
	Method Method

	// Args names the fixture fields it is handed.
	Args []string
}

// undeliverableArgs names the first argument the fixture cannot supply, with
// the field it would have come from so a caller can say why.
func undeliverableArgs(f Fixture, args []string) (string, FixtureField, bool) {
	for _, name := range args {
		field, found := f.Field(strings.TrimSuffix(name, OtherSuffix))
		if !found || !field.OK() {
			return name, field, true
		}
	}
	return "", FixtureField{}, false
}
