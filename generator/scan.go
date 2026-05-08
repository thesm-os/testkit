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

// ScanConstsOfType returns every exported package-level constant
// whose declared type is the named type, sorted by source position.
// The returned [*ConstInfo] carries the const's value, doc comment,
// and inline comment — enough for the enum generator to derive
// expected stringer output and wire-compat mappings.
//
// Sort order is source position rather than name so the wire-compat
// golden reflects the user's iota ordering. A reorder shows up as a
// reordered JSON document in the diff.
func ScanConstsOfType(pkg *Package, typeName string) []*ConstInfo {
	scope := pkg.Pkg.Scope()
	var out []*ConstInfo
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if !obj.Exported() {
			continue
		}
		c, ok := obj.(*types.Const)
		if !ok {
			continue
		}
		named, ok := c.Type().(*types.Named)
		if !ok || named.Obj().Name() != typeName {
			continue
		}
		ci, err := pkg.Const(c.Name())
		if err != nil {
			continue
		}
		out = append(out, ci)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Pos.Filename != out[j].Pos.Filename {
			return out[i].Pos.Filename < out[j].Pos.Filename
		}
		return out[i].Pos.Offset < out[j].Pos.Offset
	})
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

// HasFunc reports whether the package declares a top-level function
// with the given name whose signature satisfies sigCheck. Returns
// false when the lookup misses, the symbol isn't a function, or the
// signature doesn't match.
//
// Use this for free-function predicates like ParseEnum where the
// detection target is a package-level helper rather than a method.
func HasFunc(pkg *Package, name string, sigCheck func(*types.Signature) bool) bool {
	obj := pkg.Pkg.Scope().Lookup(name)
	if obj == nil {
		return false
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		return false
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return false
	}
	return sigCheck(sig)
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

// StringerSig matches `() string` — the canonical [fmt.Stringer]
// signature. Used by enum (and any future stub-side stringer
// detection) without dragging fmt into the dependency graph.
func StringerSig(sig *types.Signature) bool {
	if sig.Params().Len() != 0 || sig.Results().Len() != 1 {
		return false
	}
	res, ok := sig.Results().At(0).Type().Underlying().(*types.Basic)
	return ok && res.Kind() == types.String
}

// ParseSig returns a predicate matching `(string) (<typeName>, error)` —
// the canonical Parse<Type> signature for enum-style parse helpers.
// The returned predicate closes over typeName so callers don't need
// to reimplement the named-result-type check.
func ParseSig(typeName string) func(*types.Signature) bool {
	return func(sig *types.Signature) bool {
		if sig.Params().Len() != 1 || sig.Results().Len() != 2 {
			return false
		}
		b, ok := sig.Params().At(0).Type().(*types.Basic)
		if !ok || b.Kind() != types.String {
			return false
		}
		named, ok := sig.Results().At(0).Type().(*types.Named)
		if !ok || named.Obj().Name() != typeName {
			return false
		}
		return IsErrorType(sig.Results().At(1).Type())
	}
}

// MarshalTextSig matches `() ([]byte, error)` — the canonical
// [encoding.TextMarshaler] signature.
func MarshalTextSig(sig *types.Signature) bool {
	return marshalBytesErrSig(sig, 0)
}

// UnmarshalTextSig matches `([]byte) error` — the canonical
// [encoding.TextUnmarshaler] signature. Pointer receivers are
// required for unmarshal so the helper is used with [HasMethod].
func UnmarshalTextSig(sig *types.Signature) bool {
	return unmarshalBytesErrSig(sig)
}

// MarshalJSONSig matches `() ([]byte, error)` — the canonical
// [encoding/json.Marshaler] signature. Identical wire shape to
// [MarshalTextSig]; kept distinct so callers express intent.
func MarshalJSONSig(sig *types.Signature) bool {
	return marshalBytesErrSig(sig, 0)
}

// UnmarshalJSONSig matches `([]byte) error` — the canonical
// [encoding/json.Unmarshaler] signature.
func UnmarshalJSONSig(sig *types.Signature) bool {
	return unmarshalBytesErrSig(sig)
}

// MarshalBinarySig matches `() ([]byte, error)` — the canonical
// [encoding.BinaryMarshaler] signature. Identical wire shape to
// the text/JSON variants; kept distinct so callers express intent.
func MarshalBinarySig(sig *types.Signature) bool {
	return marshalBytesErrSig(sig, 0)
}

// UnmarshalBinarySig matches `([]byte) error` — the canonical
// [encoding.BinaryUnmarshaler] signature.
func UnmarshalBinarySig(sig *types.Signature) bool {
	return unmarshalBytesErrSig(sig)
}

// marshalBytesErrSig reports whether sig has paramCount params, a
// []byte first result, and an error second result.
func marshalBytesErrSig(sig *types.Signature, paramCount int) bool {
	if sig.Params().Len() != paramCount || sig.Results().Len() != 2 {
		return false
	}
	if !isByteSlice(sig.Results().At(0).Type()) {
		return false
	}
	return IsErrorType(sig.Results().At(1).Type())
}

// unmarshalBytesErrSig reports whether sig has a single []byte
// param and a single error result.
func unmarshalBytesErrSig(sig *types.Signature) bool {
	if sig.Params().Len() != 1 || sig.Results().Len() != 1 {
		return false
	}
	if !isByteSlice(sig.Params().At(0).Type()) {
		return false
	}
	return IsErrorType(sig.Results().At(0).Type())
}

// isByteSlice reports whether t is `[]byte` (slice of basic byte).
func isByteSlice(t types.Type) bool {
	s, ok := t.Underlying().(*types.Slice)
	if !ok {
		return false
	}
	b, ok := s.Elem().Underlying().(*types.Basic)
	return ok && b.Kind() == types.Byte
}

// ErrorInterface returns the builtin `error` interface as a
// [*types.Interface], cached at package init via go/types.Universe.
// Use with [ScanStructsImplementing] to find custom error types.
func ErrorInterface() *types.Interface {
	return errorIfaceCache
}

var errorIfaceCache = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
