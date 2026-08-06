// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/store"

	"go.thesmos.sh/testkit/generator/internal/samples"
)

// Sample is one value a generated check sets a field to.
//
// The value travels as a reference plus text rather than as text alone because
// anything but a builtin has to name its type, and a rendered file registers an
// import only for a reference it was handed — text cannot ask for one. The
// template composes the two.
type Sample struct {
	// Ref qualifies the type the text is written against, nil when the text
	// stands alone.
	Ref emit.Ref

	// Text is the literal. Empty when no sample could be derived, which is the
	// caller's signal to omit the check rather than emit one that cannot fail.
	Text string

	// Composite writes `Ref{Text}`; otherwise `Ref(Text)`.
	Composite bool
}

// OK reports whether a value was derived.
func (s Sample) OK() bool { return s.Text != "" }

// literal returns a sample that needs no type beside it.
func literal(text string) Sample { return Sample{Text: text} }

// resolver answers what a named type is, over the packages the run loaded.
//
// A generator that only reads field types sees `Weekday` and `Role` as opaque
// names and can write no value for either, so every setter taking one goes
// unchecked. The declarations are in the graph already — the frontend records a
// defined type's underlying type and a struct's fields — so the names are
// answerable, and only a type from a package the run never read is not.
type resolver struct {
	aliases map[string]*node.Alias
	structs map[string]*node.Struct
}

// newResolver indexes the loaded declarations by qualified name.
//
// Read through the reader rather than the store so the lookups land in the
// plugin's cache key: a sample derived from a struct that has since changed is
// stale in exactly the way a cache exists to catch.
func newResolver(r *store.Reader) *resolver {
	rv := &resolver{
		aliases: make(map[string]*node.Alias),
		structs: make(map[string]*node.Struct),
	}
	for _, a := range r.Aliases().Slice() {
		rv.aliases[qualified(a.Package, a.Name)] = a
	}
	for _, s := range r.Structs().Slice() {
		rv.structs[qualified(s.Package, s.Name)] = s
	}
	return rv
}

// qualified keys a type by package and name.
func qualified(pkg, name string) string { return pkg + "." + name }

// samples derives the pair for a source type, or two empty samples when the
// type admits none.
//
// seen carries the named types already being derived, so a struct reachable
// from itself — `Manager *User` inside `User` — terminates rather than
// recursing forever.
func (rv *resolver) samples(t *node.TypeRef, fieldName string, seen map[string]bool) (sample, alternate Sample) {
	switch {
	case t == nil:
		return Sample{}, Sample{}

	case t.TypeKind == node.TypeRefArray:
		// `[3]int{42}` differs from `[3]int{7}` in one element, which is all a
		// check needs; filling the rest would say nothing more.
		return rv.wrap(t, t.Elem, fieldName, seen, func(text string) string {
			return "{" + text + "}"
		})

	case isAny(t):
		// `any` admits every value, so the string pair serves. The conversion
		// keeps both sides of the comparison the same type, which is what lets
		// the check's type parameter be inferred.
		s, a := samples.For("string", fieldName)
		ref := emit.Builtin("any")
		return Sample{Ref: ref, Text: s}, Sample{Ref: ref, Text: a}

	case t.IsBuiltin():
		s, a := samples.For(t.Name, fieldName)
		return literal(s), literal(a)

	case t.TypeKind == node.TypeRefNamed:
		return rv.named(t, fieldName, seen)
	}
	return Sample{}, Sample{}
}

// named derives the pair for a type the graph may hold a declaration for.
func (rv *resolver) named(t *node.TypeRef, fieldName string, seen map[string]bool) (sample, alternate Sample) {
	key := qualified(t.Package, t.Name)
	if seen[key] {
		return Sample{}, Sample{}
	}
	seen[key] = true
	defer delete(seen, key)

	// A defined type keeps its own type through the setter, so its value has to
	// be written as a conversion — `Weekday(42)` rather than a bare 42, which
	// would compile here and stop compiling the moment the field's type moved.
	if a := rv.aliases[key]; a != nil {
		return rv.wrap(t, a.Target, fieldName, seen, func(text string) string { return text })
	}
	if s := rv.structs[key]; s != nil {
		return rv.composite(t, s, seen)
	}
	return Sample{}, Sample{}
}

// composite writes a struct literal setting one of the struct's own fields.
//
// The first exported field whose type yields a bare literal wins. Restricting
// the inner value to a literal is what bounds this: a nested value needing a
// type of its own would have to render that type too, and a reference cannot be
// folded into the text of another.
func (rv *resolver) composite(t *node.TypeRef, s *node.Struct, seen map[string]bool) (sample, alternate Sample) {
	for _, f := range s.Fields {
		if !golang.IsExported(f.Name) {
			continue
		}
		inner, innerAlt := rv.samples(f.Type, f.Name, seen)
		if !inner.OK() || inner.Ref != nil {
			continue
		}
		ref := golang.FromNode(t)
		return Sample{Ref: ref, Text: "{" + f.Name + ": " + inner.Text + "}", Composite: true},
			Sample{Ref: ref, Text: "{" + f.Name + ": " + innerAlt.Text + "}", Composite: true}
	}
	return Sample{}, Sample{}
}

// wrap derives the pair from inner and rewrites each value's text against t.
//
// Only a bare literal is wrapped, for the reason [resolver.composite] gives.
func (rv *resolver) wrap(t, inner *node.TypeRef, fieldName string, seen map[string]bool,
	text func(string) string,
) (sample, alternate Sample) {
	s, a := rv.samples(inner, fieldName, seen)
	if !s.OK() || s.Ref != nil {
		return Sample{}, Sample{}
	}
	ref := golang.FromNode(t)
	composite := t.TypeKind == node.TypeRefArray
	return Sample{Ref: ref, Text: text(s.Text), Composite: composite},
		Sample{Ref: ref, Text: text(a.Text), Composite: composite}
}

// isAny reports whether t is `any` written as an empty interface, which the
// frontend records as an anonymous interface declaring nothing.
func isAny(t *node.TypeRef) bool {
	return t.TypeKind == node.TypeRefAnonInterface && len(t.Methods) == 0 && len(t.Embeds) == 0
}

// samplesOfRef derives the pair from an emit reference, which is what a field's
// type has become by the time witnesses are substituted into it.
//
// Builtins only: a witness is always one, and this runs after the source types
// are gone, so there is nothing left to resolve a named type against.
func samplesOfRef(r emit.Ref, fieldName string) (sample, alternate Sample) {
	b, ok := r.(*emit.BuiltinRef)
	if !ok {
		return Sample{}, Sample{}
	}
	s, a := samples.For(b.Name, fieldName)
	return literal(s), literal(a)
}

// sampleSource returns the reference a field's pair is derived from, which is
// whatever its setter actually takes: the pointee for a pointer, the key for a
// set, and the field's own type otherwise.
func sampleSource(f Field) emit.Ref {
	switch f.Shape {
	case Pointer:
		return f.Elem
	case Set:
		return f.Key
	default:
		return f.Type
	}
}
