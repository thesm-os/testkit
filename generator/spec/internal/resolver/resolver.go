// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package resolver carries the shared cross-package symbol resolution
// logic used by directive consumers that resolve Go references in
// their args (sample, errors, wrapped-via, hooks, ...).
//
// Every Pattern C consumer (per spec/doc.go's taxonomy) follows the
// same loop:
//
//  1. Split each arg into (importPath, name, qualified).
//  2. Local: lookup name in the source pkg's scope.
//  3. Remote: load importPath via [spec.Data.Loader], lookup name
//     there, register the import on [spec.Data.Tracker].
//  4. Apply consumer-specific kind/shape validation against the
//     resolved [types.Object].
//  5. Render the final expression as `alias.Name` or `Name`.
//
// This package owns steps 1, 2, 3, 5 — every consumer reuses them.
// Step 4 (kind/shape validation) is consumer-specific and stays in
// the consumer's package.
package resolver

import (
	"errors"
	"fmt"
	"go/types"
	"strings"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
)

// SplitQualified splits arg into (importPath, name, qualified).
//
// arg is qualified when its LHS-of-last-dot contains "/" — i.e.
// looks like a Go import path. Examples:
//
//	"SampleKey"                                 → ("",                          "SampleKey", false)
//	"fixtures.SampleKey"                        → ("",        "fixtures.SampleKey", false)
//	"go.thesmos.sh/myproj/fixtures.SampleKey"   → ("go.thesmos.sh/myproj/fixtures", "SampleKey", true)
//
// The middle case (dotted but no slash) is treated as local — a
// future enhancement could resolve it via the source file's
// existing import declarations.
func SplitQualified(arg string) (importPath, name string, qualified bool) {
	dot := strings.LastIndex(arg, ".")
	if dot < 0 {
		return "", arg, false
	}
	lhs := arg[:dot]
	if !strings.Contains(lhs, "/") {
		return "", arg, false
	}
	return lhs, arg[dot+1:], true
}

// Resolved is the outcome of a successful symbol lookup.
type Resolved struct {
	// Alias is the import-tracker alias for the resolved package
	// when the reference is qualified. Empty for local references.
	Alias string

	// Name is the bare symbol name (no package qualifier).
	Name string

	// Obj is the resolved [types.Object]. Consumer applies its
	// own kind/shape validation.
	Obj types.Object
}

// Render returns the call/reference expression a template emits:
// "Alias.Name" when Alias is non-empty, else "Name".
func (r Resolved) Render() string {
	if r.Alias == "" {
		return r.Name
	}
	return r.Alias + "." + r.Name
}

// Resolve looks up arg as either a local-package or remote-package
// symbol. The remote branch loads via [spec.Data.Loader] (cache
// hits accumulate across calls in the same Enrich pass) and
// registers the import on [spec.Data.Tracker] so the rendered
// expression's alias matches the rest of the file's imports.
//
// Either branch consults the tracker for the resolved package's
// alias — local symbols get qualified ("basic.ErrNotFound") when
// the output file lives in a different package from the source,
// bare ("ErrNotFound") otherwise. The tracker's [AddPath] returns
// "" for the local-to-output package, which the renderer treats as
// "no qualifier needed."
//
// Returns a [Resolved] on success. Errors are returned with a
// short message; consumers wrap them with their own directive-named
// prefix.
func Resolve(arg string, data *spec.Data, pkg *generator.Package) (Resolved, error) {
	importPath, name, qualified := SplitQualified(arg)
	if !qualified {
		return resolveLocal(name, data, pkg)
	}
	return resolveRemote(importPath, name, data)
}

// resolveLocal looks up name in pkg's top-level scope and consults
// data.Tracker to render the appropriate alias. When the output
// file lives in pkg itself, the tracker returns "" and the rendered
// expression stays bare; when output is in a sibling package, the
// tracker returns the source pkg's alias ("basic") and the rendered
// expression carries it.
func resolveLocal(name string, data *spec.Data, pkg *generator.Package) (Resolved, error) {
	obj := pkg.Pkg.Scope().Lookup(name)
	if obj == nil {
		return Resolved{}, fmt.Errorf("symbol not found in package %s", pkg.Path())
	}
	var alias string
	if data != nil && data.Tracker != nil {
		alias = data.Tracker.AddPath(pkg.Path())
	}
	return Resolved{Alias: alias, Name: name, Obj: obj}, nil
}

// resolveRemote loads importPath via the shared loader, looks up
// name in the resolved package, and registers the import on the
// tracker. Returns the alias from the tracker for rendering.
func resolveRemote(importPath, name string, data *spec.Data) (Resolved, error) {
	if data.Loader == nil {
		return Resolved{}, errors.New(
			"resolver: cross-package reference requires Data.Loader (set by spec.Analyze)",
		)
	}
	remote, err := data.Loader.Load(importPath, "")
	if err != nil {
		return Resolved{}, fmt.Errorf("load %s: %w", importPath, err)
	}
	obj := remote.Pkg.Scope().Lookup(name)
	if obj == nil {
		return Resolved{}, fmt.Errorf("symbol not found in package %s", importPath)
	}
	alias := data.Tracker.AddPath(importPath)
	// AddPath returns "" for the local package; an arg pointing at
	// the source pkg from the qualified branch shouldn't happen,
	// but fall back to the bare name to avoid an "x." prefix.
	return Resolved{Alias: alias, Name: name, Obj: obj}, nil
}

// RequireArgs validates the directive's argument count against want
// and returns a uniform diagnostic on mismatch.
//
// Used by consumers whose arg count is known a priori (Sample's
// "one per non-ctx param", Errors' "one or more sentinels"). For
// variable counts the consumer applies its own check.
func RequireArgs(dir directive.Directive, want int) error {
	if len(dir.Args) != want {
		return fmt.Errorf("expects %d arg(s), got %d", want, len(dir.Args))
	}
	return nil
}

// FuncSig describes the expected shape of a function symbol so
// consumers can validate that a resolved [types.Object] is a
// function with the right parameter and result types.
//
// Each field is set by the consumer; absent fields default to
// "must be empty":
//
//	sample:    FuncSig{Results: []types.Type{paramType}}            // func() T
//	hooks:     FuncSig{Params:  []types.Type{ctxType, eventType},   // func(ctx, T) error
//	                  Results: []types.Type{errorType}}
//	defaults:  FuncSig{Results: []types.Type{namedType}}            // func() T
//
// Type comparison uses [types.Identical], so type-parameter
// resolution and named-type identity are honored.
type FuncSig struct {
	// Params is the expected parameter types in order. nil/empty
	// means the func must take no parameters.
	Params []types.Type

	// Results is the expected result types in order. nil/empty
	// means the func must have no results.
	Results []types.Type

	// Variadic, when true, requires the last parameter to be a
	// variadic of [Params]'s last entry's element type. False
	// rejects variadic signatures.
	Variadic bool
}

// Check validates obj matches the [FuncSig]. Returns nil on match;
// returns a diagnostic naming the first mismatch otherwise.
//
// Diagnostic style stays compact ("not a function", "param 0:
// returns X, expected Y") so consumers can wrap with their own
// directive-named prefix without nesting verbose context.
func (s FuncSig) Check(obj types.Object) error {
	fn, ok := obj.(*types.Func)
	if !ok {
		return fmt.Errorf("not a function (got %T)", obj)
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return errors.New("resolver: not a signature")
	}
	if sig.Variadic() != s.Variadic {
		if s.Variadic {
			return errors.New("resolver: must be variadic")
		}
		return errors.New("resolver: must not be variadic")
	}
	if got := sig.Params().Len(); got != len(s.Params) {
		return fmt.Errorf("expects %d param(s), got %d", len(s.Params), got)
	}
	for i, want := range s.Params {
		if got := sig.Params().At(i).Type(); !types.Identical(got, want) {
			return fmt.Errorf("param %d: %s, expected %s", i,
				types.TypeString(got, nil), types.TypeString(want, nil))
		}
	}
	if got := sig.Results().Len(); got != len(s.Results) {
		return fmt.Errorf("expects %d result(s), got %d", len(s.Results), got)
	}
	for i, want := range s.Results {
		if got := sig.Results().At(i).Type(); !types.Identical(got, want) {
			return fmt.Errorf("result %d: %s, expected %s", i,
				types.TypeString(got, nil), types.TypeString(want, nil))
		}
	}
	return nil
}

// VarOfType validates obj is a [types.Var] (package-level variable
// or constant) whose type is assignable to want. Used by consumers
// that resolve sentinel error variables (errors, wrapped-via): the
// arg references an exported var declared as a sentinel error.
//
// Assignability (not identity) lets `var ErrFoo = errors.New(...)`
// resolve under want = the builtin error interface — *errors.errorString
// is assignable to error, not identical to it.
func VarOfType(obj types.Object, want types.Type) error {
	v, ok := obj.(*types.Var)
	if !ok {
		// Package-level variables can also surface as
		// *types.Const (rare) or other Object kinds. The
		// canonical sentinel form is `var ErrX = ...` which
		// produces *types.Var.
		return fmt.Errorf("not a variable (got %T)", obj)
	}
	if !types.AssignableTo(v.Type(), want) {
		return fmt.Errorf("type %s not assignable to %s",
			types.TypeString(v.Type(), nil),
			types.TypeString(want, nil))
	}
	return nil
}

// ErrorType returns the builtin `error` interface — the canonical
// `want` for [VarOfType] when validating sentinel error variables.
func ErrorType() types.Type {
	return types.Universe.Lookup("error").Type()
}
