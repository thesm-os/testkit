// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"strings"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
)

// samplesFor returns two distinct values of the named Go builtin as source
// text, or two empty strings when the type admits none.
//
// Two values rather than one, because a check comparing a field against a
// single value passes whenever the constructor already seeded that value — and
// the seed is not always knowable here, since a companion's return is opaque to
// this generator. Whatever it seeded equals at most one of a pair, so a setter
// that assigns nothing fails against the other.
//
// The string form carries the field's own name so a value in a failure message
// says which setter produced it.
//
// Returned as source text rather than as a reference because every value is a
// builtin literal: it names no package, so nothing here can produce something
// the rendered file would have to import. That is the same reason
// [witness.Args] may return a string.
//
// Only builtins are answered, and the two omissions are deliberate. A defined
// type is recorded by name — `Weekday int` arrives as `Weekday` with no way to
// learn it is an integer — so a literal written for it would not compile. A
// struct, interface, channel, function or array has no literal this generator
// could write that differs from its zero value, which is exactly the vacuity
// the pair exists to prevent; the caller omits the check rather than emitting
// one that cannot fail.
// The arms spell Go's own type names, which several unrelated tables in this
// module also spell — witness's palette, the double's no-import set. They answer
// different questions about one domain rather than sharing an answer, so
// hoisting the names to constants would put an identifier between each table and
// the thing it is about, and is suppressed instead.
//
//nolint:goconst // see above.
func samplesFor(typeName, fieldName string) (sample, alternate string) {
	switch typeName {
	case "string":
		lower := strings.ToLower(fieldName)
		return `"test-` + lower + `"`, `"other-` + lower + `"`
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"byte", "rune":
		return "42", "7"
	case "float32", "float64":
		return "3.14", "2.72"
	case "bool":
		// The only type whose pair exhausts its values, which is what makes the
		// bool check the strictest of them: a setter assigning nothing fails
		// against one arm no matter what the constructor seeded.
		return "true", "false"
	}
	return "", ""
}

// samplesOfNode derives the pair from a source type reference.
//
// A type parameter is also a package-less named reference, so it reaches
// [samplesFor] and yields nothing there; the pair for a parameterised field is
// derived instead once substitution resolves it to a witness.
func samplesOfNode(t *node.TypeRef, fieldName string) (sample, alternate string) {
	if t == nil || !t.IsBuiltin() {
		return "", ""
	}
	return samplesFor(t.Name, fieldName)
}

// samplesOfRef derives the pair from an emit reference, which is what a field's
// type has become by the time witnesses are substituted into it.
func samplesOfRef(r emit.Ref, fieldName string) (sample, alternate string) {
	b, ok := r.(*emit.BuiltinRef)
	if !ok {
		return "", ""
	}
	return samplesFor(b.Name, fieldName)
}

// sampleSource returns the reference a field's pair is derived from, which is
// whatever its setter actually takes: the pointee for a pointer, the key for a
// set, and the field's own type otherwise.
//
// A slice, byte slice or map lands in the last arm and yields nothing, since
// none of them is a builtin reference — which is correct, as their checks
// assert on length and on merge-versus-replace and are already unable to pass
// against a setter that does nothing.
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
