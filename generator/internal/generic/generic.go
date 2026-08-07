// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package generic renders a parameterised declaration's type parameters.
//
// Two spellings are needed wherever a generator emits code beside a generic
// type: the declaration form `[K comparable, V any]`, and the use form
// `[K, V]`. Every identifier a generated file declares for its subject has to
// carry both in the right places, and a generic type referenced bare does not
// compile.
//
// Taking the parameter list rather than the declaration that holds it is what
// makes one implementation serve interfaces and structs alike — the container
// has no bearing on the answer, and keying on it is what produced three copies
// of this that had to be kept in step.
package generic

import (
	"strings"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
)

// Params lifts the source's type parameters into emit form, which is what the
// backend's `renderTypeParams` spells as `[K comparable, V any]`.
//
// Returns nil for a declaration carrying none, so a template appending the
// rendered form does so unconditionally.
func Params(params []*node.TypeParam) []*emit.TypeParam {
	if len(params) == 0 {
		return nil
	}
	out := make([]*emit.TypeParam, len(params))
	for i, p := range params {
		out[i] = &emit.TypeParam{
			Name:       p.Name,
			Constraint: golang.ConstraintFromNode(p.Constraint),
		}
	}
	return out
}

// Args renders the same list in use position — `[K, V]` — or empty for a
// declaration carrying none.
//
// Safe to build as a string because a type parameter's use form is its own
// name: it names no package, so nothing here can produce something the rendered
// file would have to import. That is not true of [Params], whose constraints
// can name one, which is why that half goes through emit and this half does
// not.
func Args(params []*node.TypeParam) string {
	if len(params) == 0 {
		return ""
	}
	names := make([]string, len(params))
	for i, p := range params {
		names[i] = p.Name
	}
	return "[" + strings.Join(names, ", ") + "]"
}
