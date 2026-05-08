// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directive

import "go/token"

// Directive is a parsed //testkit:<name> [args...] annotation. The
// parser ([ParseLine]) extracts directives from doc comments;
// generators attach the resulting slices to their analyzed types
// (methods, fields, structs, vars, consts).
//
// Directive lives in this package — not in the generator core — so
// the directive registry, validator, composition logic, and emitter
// dispatch can all operate without depending on the generator package
// (which would create an import cycle).
type Directive struct {
	// Name is the directive name without the //testkit: prefix
	// ("errors", "concurrent", "sample", ...).
	Name string

	// Args are the directive's arguments. For standalone lines they
	// are space-split; for tokens inside a `//testkit:directive`
	// bundle they are comma-split off the value side of the "=".
	//
	//   //testkit:errors ErrA ErrB                → ["ErrA", "ErrB"]
	//   //testkit:directive errors=ErrA,ErrB      → ["ErrA", "ErrB"]
	Args []string

	// Off is true when the source declared an opt-out form via the
	// bundle syntax: "//testkit:directive mutator=off". The opt-out
	// semantics are directive-specific — shape detectors and mixin
	// emitters check Off to suppress their behavior. Off=true
	// directives carry no Args by convention.
	Off bool

	// Pos is the source position of the directive line (best-effort
	// — the loader uses the declaration's position when go/ast does
	// not preserve per-line positions of doc-comment text).
	Pos token.Position
}
