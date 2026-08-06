// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package gotmpl supplies the Go-rendering helpers testkit's generator
// templates call.
//
// Every generator that renders a method has to spell the same lists: the
// argument list of a call, the field assignments of a recorded call, the
// locals a delegate's result is captured into. Written as template `define`
// blocks those lists are duplicated per generator, and the duplicates are
// silent — a copy that picks the wrong return slot generates code that
// compiles and asserts the wrong thing.
//
// Written here they are ordinary Go functions: unit-tested, compile-checked,
// and shared without either generator naming the other's templates. The
// backend merges every plugin's template `define` names into one tree
// (backend/golang merge.go), so a shared define would make one plugin's
// private template names another plugin's API — a rename would break output
// at render time rather than at build time.
//
// # What is not here
//
// Anything that renders a type. `renderType` is bound to the backend's render
// state because it registers the file's imports and elides same-package
// qualifiers; a free function cannot reproduce either, and one that formatted
// types itself would name packages the rendered file never imports. Type-
// bearing lists therefore stay as template defines, private to each generator.
//
// # Hazards
//
// Every function here builds a string with no knowledge of the target file,
// so nothing it returns can introduce an import. Callers pass the projection
// from [signature], never a hand-built slice: the field, local, and error-slot
// conventions are that package's, and a caller that reconstructs them is the
// duplication this package exists to remove.
package gotmpl

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"go.thesmos.sh/eidos/core/naming"
	"go.thesmos.sh/eidos/emit"

	"go.thesmos.sh/testkit/generator/internal/signature"
)

// Module is testkit's module path.
//
// Generated code references the runtime by import path, and a path spelled in
// each template would have to be corrected in every one of them the day the
// module moves. Held here so the templates name a package rather than a path.
const Module = "go.thesmos.sh/testkit"

// packages maps the short alias a template writes to the import path it
// resolves to. The set is closed deliberately: an alias that is not here is a
// typo, and reporting it is worth more than resolving whatever was written.
//
//nolint:gochecknoglobals // immutable lookup table.
var packages = map[string]string{
	"testkit": Module,
	"clock":   Module + "/clock",
	"fault":   Module + "/fault",
	"rand":    Module + "/rand",
	"stub":    Module + "/stub",
	"errors":  "errors",
	"fmt":     "fmt",
	"strings": "strings",
	"testing": "testing",
	"time":    "time",
}

// ErrBadSymbol is returned by [Ref] for a symbol that is not
// `<package>.<Symbol>` or that names a package outside [packages].
//
// It is an error rather than a fallback because text/template aborts the
// render on it: a mistyped symbol fails the build loudly instead of silently
// emitting a reference to a package the file never imports.
var ErrBadSymbol = errors.New("gotmpl: bad symbol")

// Ref resolves a `<package>.<Symbol>` shorthand to the qualified reference the
// backend renders — `Ref("testkit.ErrorIs")` for `testkit.ErrorIs`.
//
// The package half is an alias from [packages], not an import path. That is
// the point: a template says which package it means, and the path each alias
// resolves to lives in one place.
func Ref(symbol string) (*emit.Expr, error) {
	alias, name, split := strings.Cut(symbol, ".")
	if !split || alias == "" || name == "" {
		return nil, fmt.Errorf("%w: %q is not <package>.<Symbol>", ErrBadSymbol, symbol)
	}
	path, known := packages[alias]
	if !known {
		return nil, fmt.Errorf(
			"%w: %q names no known package (have: %s)",
			ErrBadSymbol, symbol, strings.Join(slices.Sorted(maps.Keys(packages)), ", "),
		)
	}
	return emit.NewExternal(path, name), nil
}

// Idents renders n positional identifiers under prefix — `Idents("a", 2)`
// gives `a0, a1`.
//
// Positional rather than named because these stand for values a generated
// check never inspects by name: the arguments a dispatch test passes, the
// results it boxes. A name would suggest a meaning the value does not carry.
func Idents(prefix string, n int) string {
	return join(n, func(i int) string { return prefix + strconv.Itoa(i) })
}

// Blanks renders n discards — `_, _`.
func Blanks(n int) string {
	return join(n, func(int) string { return "_" })
}

// Args renders the parameters as an argument list — `ctx, id`, or
// `ctx, keys...` where the tail is variadic.
//
// The ellipsis is not decoration: forwarding a variadic parameter without it
// passes the slice as a single element, which type-checks against `...any` and
// silently records one argument where the caller passed several.
func Args(params []signature.Param) string {
	return join(len(params), func(i int) string {
		if params[i].Variadic {
			return params[i].Name + "..."
		}
		return params[i].Name
	})
}

// IdentArgs renders positional identifiers as an argument list, spreading a
// variadic tail — `IdentArgs("a", …)` gives `a0, a1...`.
//
// Separate from [Idents] because a call site needs the spread and a
// declaration list must not have it.
func IdentArgs(prefix string, params []signature.Param) string {
	return join(len(params), func(i int) string {
		ident := prefix + strconv.Itoa(i)
		if params[i].Variadic {
			return ident + "..."
		}
		return ident
	})
}

// CallFields renders the parameters as recorded-call field assignments —
// `Ctx: ctx, ID: id`.
func CallFields(params []signature.Param) string {
	return join(len(params), func(i int) string {
		return params[i].Field + ": " + params[i].Name
	})
}

// Locals renders the capture identifiers a delegate's results bind to —
// `r0, r1`, or the source's own names where the signature declares them.
func Locals(returns []signature.Return) string {
	return join(len(returns), func(i int) string { return returns[i].Local })
}

// LocalFields renders the return tuple built from those captures —
// `Result: r0, Err: r1`.
func LocalFields(returns []signature.Return) string {
	return join(len(returns), func(i int) string {
		return returns[i].Field + ": " + returns[i].Local
	})
}

// IdentFields renders the return tuple built from positional identifiers —
// `IdentFields("got", …)` gives `Result: got0, Err: got1`.
func IdentFields(prefix string, returns []signature.Return) string {
	return join(len(returns), func(i int) string {
		return returns[i].Field + ": " + prefix + strconv.Itoa(i)
	})
}

// NamedFields renders the return tuple built from the consumer-facing
// parameter names — `Result: result, Err: err`.
//
// Named after the recorded-call fields rather than after the internal capture
// locals: this is the surface a caller of the generated Returns setter reads.
func NamedFields(returns []signature.Return) string {
	return join(len(returns), func(i int) string {
		return returns[i].Field + ": " + naming.Camel(returns[i].Field)
	})
}

// Reads renders the return tuple being read back off a resolved answer —
// `r.Result, r.Err`.
func Reads(returns []signature.Return) string {
	return join(len(returns), func(i int) string { return "r." + returns[i].Field })
}

// Fails renders an assignment list binding only the error slot and discarding
// the rest — `_, err`.
//
// The error slot is found by flag rather than by position. A signature
// returning `(error, string)` is unusual but legal Go, and a positional rule
// would bind the wrong local without failing to compile.
func Fails(returns []signature.Return) string {
	return join(len(returns), func(i int) string {
		if returns[i].Error {
			return returns[i].Local
		}
		return "_"
	})
}

// join renders n comma-separated entries produced by at.
//
// The separator is fixed at ", " because every list here is a Go expression
// list, and the backend runs gofmt over the result anyway — a caller choosing
// its own separator would be choosing formatting the formatter overrides.
func join(n int, at func(i int) string) string {
	parts := make([]string, n)
	for i := range n {
		parts[i] = at(i)
	}
	return strings.Join(parts, ", ")
}

// FuncMap returns the helpers above under prefix, for a plugin to contribute
// through TemplateFuncs.
//
// Prefixed rather than shared under one name because the backend rejects two
// plugins registering the same extension outright (ErrTemplateFuncCollision,
// backend/golang merge.go). Sharing the logic in Go and the namespace not at
// all is what lets a second generator contribute into a first one's output
// without either becoming the other's API.
func FuncMap(prefix string) template.FuncMap {
	return template.FuncMap{
		prefix + "Args":        Args,
		prefix + "Blanks":      Blanks,
		prefix + "CallFields":  CallFields,
		prefix + "Fails":       Fails,
		prefix + "IdentArgs":   IdentArgs,
		prefix + "IdentFields": IdentFields,
		prefix + "Idents":      Idents,
		prefix + "LocalFields": LocalFields,
		prefix + "Locals":      Locals,
		prefix + "NamedFields": NamedFields,
		prefix + "Reads":       Reads,
		prefix + "Ref":         Ref,
	}
}
