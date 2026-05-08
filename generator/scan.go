// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator

import (
	"go/types"
	"path/filepath"
	"sort"
)

// ScanVars walks the package's exported package-level variables,
// returning every [*VarInfo] for which keep returns true, sorted by
// name. When sourceFile is non-empty the scan is restricted to vars
// declared in that file (the file-scoped $GOFILE generation path).
//
// Generators use this with a name-prefix predicate (sentinel filters
// to "Err"-prefixed) or a type-shape predicate (e.g. "any var typed
// as a custom struct").
func ScanVars(pkg *Package, sourceFile string, keep func(*types.Var) bool) []*VarInfo {
	var out []*VarInfo
	scope := pkg.Pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if !obj.Exported() {
			continue
		}
		v, ok := obj.(*types.Var)
		if !ok || !keep(v) {
			continue
		}
		if sourceFile != "" {
			pos := pkg.Fset.Position(obj.Pos())
			if filepath.Base(pos.Filename) != sourceFile {
				continue
			}
		}
		out = append(out, &VarInfo{
			Name: v.Name(),
			Type: v.Type(),
			Pos:  pkg.Fset.Position(obj.Pos()),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ScanStructsImplementing returns every exported struct type whose
// pointer receiver implements iface, sorted by name. Resolves each
// match to a fully-populated [*StructInfo] via [Package.Struct].
//
// Pointer-receiver detection is the canonical Go idiom for protocol
// satisfaction: error types, fmt.Stringer impls, etc. almost always
// have pointer-receiver methods so they remain addressable when
// passed to errors.As, fmt.Sprint, and similar.
func ScanStructsImplementing(pkg *Package, iface *types.Interface) []*StructInfo {
	scope := pkg.Pkg.Scope()
	var out []*StructInfo
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if !obj.Exported() {
			continue
		}
		named, ok := obj.Type().(*types.Named)
		if !ok {
			continue
		}
		if _, ok := named.Underlying().(*types.Struct); !ok {
			continue
		}
		if !types.Implements(types.NewPointer(named), iface) {
			continue
		}
		s, err := pkg.Struct(name)
		if err != nil {
			continue
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// HasMethod reports whether the named type has a pointer-receiver
// method with the given name whose signature satisfies sigCheck.
// Returns false for missing types, non-named types, or no matching
// method.
//
// Generators compose specialized predicates from this:
//
//	hasIs := HasMethod(pkg, "MyErr", "Is", IsErrorBoolSig)
//	hasUnwrap := HasMethod(pkg, "MyErr", "Unwrap", UnwrapSig)
func HasMethod(pkg *Package, typeName, methodName string, sigCheck func(*types.Signature) bool) bool {
	obj := pkg.Pkg.Scope().Lookup(typeName)
	if obj == nil {
		return false
	}
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return false
	}
	mset := types.NewMethodSet(types.NewPointer(named))
	for sel := range mset.Methods() {
		fn, ok := sel.Obj().(*types.Func)
		if !ok || fn.Name() != methodName {
			continue
		}
		sig := fn.Type().(*types.Signature)
		if sigCheck(sig) {
			return true
		}
	}
	return false
}

// IsErrorBoolSig matches `(error) bool` — the canonical signature
// of a stdlib `Is` method on custom error types.
func IsErrorBoolSig(sig *types.Signature) bool {
	if sig.Params().Len() != 1 || sig.Results().Len() != 1 {
		return false
	}
	if !IsErrorType(sig.Params().At(0).Type()) {
		return false
	}
	res, ok := sig.Results().At(0).Type().Underlying().(*types.Basic)
	return ok && res.Kind() == types.Bool
}

// UnwrapSig matches `() error` — the canonical signature of a
// stdlib `Unwrap` method on custom error types.
func UnwrapSig(sig *types.Signature) bool {
	return sig.Params().Len() == 0 &&
		sig.Results().Len() == 1 &&
		IsErrorType(sig.Results().At(0).Type())
}

// ErrorInterface returns the builtin `error` interface as a
// [*types.Interface], cached at package init via go/types.Universe.
// Use with [ScanStructsImplementing] to find custom error types.
func ErrorInterface() *types.Interface {
	return errorIfaceCache
}

var errorIfaceCache = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
