// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
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

	// Sample and Other are the two derived values. Other is derived to differ
	// from Sample rather than trusted to.
	Sample, Other golang.Sample

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

// OtherName is the identifier of the companion field.
func (f FixtureField) OtherName() string { return f.Name + OtherSuffix }

// OK reports whether a value for this field could be produced at all — a
// companion, or both halves of a derived pair.
//
// A parameter whose type admits no literal and declares no companion — a
// channel, a func, a type from a package the run never read — yields neither,
// and the one check whose meaning is the value is dropped rather than emitted
// against something nobody could write.
func (f FixtureField) OK() bool { return f.Companion != nil || (f.Sample.OK() && f.Other.OK()) }

// Fixture is the derived input set for one interface.
type Fixture struct {
	// TypeName is the generated struct's identifier — `<Iface>Fixture`.
	TypeName string

	// CtorName is the identifier of the function returning the derived values,
	// which a consumer reads to see what they would be overriding.
	CtorName string

	Fields []FixtureField
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
// Keyed by the parameter's exported field name rather than per method, because
// a `key string` on the reader and a `key string` on the deleter are the same
// value as far as a conformance run is concerned — and giving them separate
// fields would let a consumer override one and silently not the other.
//
// A name carried by two parameters of different types is a collision the
// generated struct cannot hold. It is reported rather than resolved: renaming
// one field would produce a fixture whose names no longer match the source, and
// picking a winner would silently check one method against the other's value.
func fixtureOf(ctx *sdk.GeneratorContext, iface *sdk.Interface, methods []Method) Fixture {
	f := Fixture{
		TypeName: iface.Name + "Fixture",
		CtorName: "Default" + iface.Name + "Fixture",
	}
	seen := map[string]golang.Param{}
	for _, m := range methods {
		for _, p := range m.CallArgs() {
			prior, dup := seen[p.Field]
			if dup {
				// [node.TypeRef.Equal] rather than comparing QNames. QName is
				// empty for every composite ref, so `[]byte` and `[]string`
				// compared equal and the diagnostic could never fire for
				// exactly the shapes it exists for. Equal is structural.
				if !prior.Source.Equal(p.Source) {
					ctx.Diag.Errorf(iface.Pos(),
						"%s: interface %q takes %q as %s in one method and %s in another; "+
							"the derived fixture cannot hold both, so rename one parameter",
						Name, iface.Name, p.Field,
						golang.Display(prior.Source), golang.Display(p.Source))
				}
				continue
			}
			seen[p.Field] = p

			sample, other := sampleFor(p, ctx.Reader)
			f.Fields = append(f.Fields, FixtureField{
				Name:      p.Field,
				Type:      p.Type,
				Sample:    sample,
				Other:     other,
				Companion: companionFor(ctx, p.Source),
			})
		}
	}
	return f
}

// withDerivableChecks drops every check whose inputs the fixture could not
// derive, and the methods left with none.
//
// A check handed a value that cannot be written down is a check that does not
// compile, and one handed the zero value where a real one was meant passes
// against a subject that does nothing. Dropping is the honest answer, and the
// generated file names each absence with the parameter that caused it — the way
// builder already explains a setter it could not check.
func withDerivableChecks(
	ctx *sdk.GeneratorContext, iface *sdk.Interface, f Fixture, methods []Method,
) []Method {
	out := make([]Method, 0, len(methods))
	for _, m := range methods {
		kept := make([]*Check, 0, len(m.Checks))
		for _, c := range m.Checks {
			if missing, ok := undeliverable(f, c); ok {
				ctx.Diag.Warnf(iface.Pos(),
					"%s: %s.%s takes %s, which no literal can be written for, so its "+
						"%q check is not generated; supply one through %sWithFixture "+
						"and write the check as %sOn%s",
					Name, iface.Name, m.Name, missing, c.Subtest,
					iface.Name, iface.Name, m.Name)
				continue
			}
			kept = append(kept, c)
		}
		// No emptiness guard: smoke is emitted for every method and needs no
		// derived input, so a method always keeps at least one check. A guard
		// here would be a branch no input can reach.
		m.Checks = kept
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
func undeliverable(f Fixture, c *Check) (string, bool) {
	if !c.NeedsDerivedInput {
		return "", false
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
	decl, resolved := r.Resolve(p.Source)
	s, ok := decl.(*sdk.Struct)
	if !resolved || !ok {
		return golang.SampleRefFor(p.Source, p.Name, r)
	}

	ref := golang.FromNode(p.Source)
	var one, other []string
	for _, f := range golang.ExportedFields(s) {
		inner, innerAlt := golang.SampleRefFor(f.Type, f.Name, r)
		if !inner.OK() {
			// A field no literal can be written for is left at its zero rather
			// than losing the whole sample: the fields around it still
			// discriminate, and refusing here would drop every check the
			// parameter feeds.
			continue
		}
		one = append(one, f.Name+": "+inner.Text)
		other = append(other, f.Name+": "+innerAlt.Text)
	}
	if len(one) == 0 {
		return golang.Sample{}, golang.Sample{}
	}
	return golang.Sample{Ref: ref, Text: "{" + strings.Join(one, ", ") + "}", Composite: true},
		golang.Sample{Ref: ref, Text: "{" + strings.Join(other, ", ") + "}", Composite: true}
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
		if shape.Get(m.Source.Meta()) != writer.Name || !m.ReturnsError() {
			continue
		}
		args := fixtureArgs(m, false)
		if missing, ok := undeliverableArgs(f, args); ok {
			_ = missing
			continue
		}
		return &Seed{Method: m, Args: args}
	}
	return nil
}

// Seed is the write a harness populates each fresh subject with.
type Seed struct {
	// Method is the writer the seed calls.
	Method Method

	// Args names the fixture fields it is handed.
	Args []string
}

// undeliverableArgs names the first argument the fixture cannot supply.
func undeliverableArgs(f Fixture, args []string) (string, bool) {
	for _, name := range args {
		field, found := f.Field(strings.TrimSuffix(name, OtherSuffix))
		if !found || !field.OK() {
			return name, true
		}
	}
	return "", false
}
