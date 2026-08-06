// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package witness picks the concrete types a generated check instantiates a
// generic subject at.
//
// A Go test function cannot take type parameters, so a check for anything
// generic lives in a generic helper that a concrete entry point calls. The
// types that entry point names have to come from somewhere, and for the two
// bounds whose type set is knowable without reading anything — `any` admits
// every type, `comparable` every basic one — they can be derived.
//
// Shared because every generator emitting checks for a generic subject faces
// the same question, and a second derivation would be free to disagree about
// which types are safe.
package witness

import (
	"strings"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
)

// Palette supplies the derived witness for each type parameter whose
// constraint admits any basic type.
//
// Positional and pairwise distinct, so a template that crossed two type
// parameters produces code that does not compile rather than code that
// typechecks and asserts the wrong thing. A subject with more parameters than
// there are entries has no derived witness at all — wrapping the list would
// hand two parameters the same type and lose exactly that property.
//
//nolint:gochecknoglobals // immutable lookup table.
var Palette = []string{"string", "int", "bool", "float64", "int64", "uint", "uint8", "int32"}

// For returns a witness per type parameter, or nil when any of them carries a
// constraint that cannot be reasoned about.
//
// All-or-nothing because an entry point instantiates the whole list at once: a
// witness for one parameter is worth nothing without one for the rest. A nil
// result is the caller's signal to emit a note in place of the checks — there
// is no way to name the types they would run at.
func For(params []*node.TypeParam) []emit.Ref {
	if len(params) == 0 || len(params) > len(Palette) {
		return nil
	}
	out := make([]emit.Ref, 0, len(params))
	for i, p := range params {
		if !admitsAnyBasicType(p.Constraint) {
			return nil
		}
		out = append(out, emit.Builtin(Palette[i]))
	}
	return out
}

// Args returns the witnesses in use position — `[string, int]` — or empty when
// there are none, so a template can append it unconditionally.
//
// Safe to build as a string because every witness is a Go builtin: its
// rendered form is its name, so nothing here can name a package the rendered
// file would need to import.
func Args(params []*node.TypeParam) string {
	refs := For(params)
	if len(refs) == 0 {
		return ""
	}
	names := make([]string, len(refs))
	for i := range refs {
		names[i] = Palette[i]
	}
	return "[" + strings.Join(names, ", ") + "]"
}

// admitsAnyBasicType reports whether every entry of [Palette] satisfies the
// constraint.
//
// Matched on the constraint's printed source form, which the frontend always
// populates, because the structured predicates do not answer this on their
// own: [node.Constraint.IsAny] holds only for a parameter with no bound written
// at all, and one written `[V any]` carries `any` as an embedded bound and so
// reads as constrained.
//
// The set is closed to the two bounds whose type set is known without loading
// anything. A named constraint is a reference into a package the generator
// never read, so a subject bounded by one takes its witnesses from the source
// or not at all.
func admitsAnyBasicType(c *node.Constraint) bool {
	if c.IsAny() || c.IsComparable() {
		return true
	}
	switch strings.TrimSpace(c.Raw) {
	case "any", "interface{}", "comparable":
		return true
	}
	return false
}
