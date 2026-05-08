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

// DefaultsFuncSig returns a predicate matching `() <typeName>` —
// the convention-based factory shape used by the builder generator
// to seed a builder with non-zero values via a sibling
// `<TypeName>Defaults()` function. Closes over typeName so callers
// don't need to reimplement the named-result-type check.
func DefaultsFuncSig(typeName string) func(*types.Signature) bool {
	return func(sig *types.Signature) bool {
		if sig.Params().Len() != 0 || sig.Results().Len() != 1 {
			return false
		}
		named, ok := sig.Results().At(0).Type().(*types.Named)
		return ok && named.Obj().Name() == typeName
	}
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

// ConcreteType is one candidate the generator can substitute for a
// generic type parameter when emitting concrete instantiations in
// generated test code. Constraint-aware selection (see
// [SelectConcreteType]) picks the first candidate whose underlying
// Go type satisfies the type parameter's constraint, so a `T any`
// param gets `string`, a `T Numeric` (~int | ~float) param gets
// `int`, and a `T comparable` param gets `string`.
type ConcreteType struct {
	// Name is the rendered Go type name as it appears in source —
	// "string", "int", "float64", etc.
	Name string

	// Type is the [types.Type] used by [types.Satisfies] to check
	// whether this candidate is in a type-parameter constraint's
	// type set.
	Type types.Type

	// Sample renders a non-zero literal of the candidate type for
	// use in generated test bodies. fieldName is consulted so
	// string samples can carry a hint ("test-id" instead of a
	// generic placeholder).
	Sample func(fieldName string) string
}

// DefaultConcreteTypes is the canonical candidate list for
// substituting into generic type parameters in generated tests.
// Order matters: [SelectConcreteType] walks from a rotated start
// so multi-parameter generics under `any`-shaped constraints get
// distinct samples per position (Pair[A,B any] → A=string, B=int);
// narrow constraints (Numeric ~int|~int64|~float64) fall through
// to the first satisfying candidate regardless of position.
//
// Covers every Go basic kind that can be a meaningful field value
// (excludes untyped kinds and unsafe.Pointer): string, bool, the
// signed/unsigned integer family, float32/64, and the complex
// pair. Strings come first because most fixtures default to a
// readable "test-<name>" sample; numerics next because they're the
// next-most-common constraint surface.
var DefaultConcreteTypes = []ConcreteType{
	{
		Name:   "string",
		Type:   types.Typ[types.String],
		Sample: func(fieldName string) string { return `"test-` + lowerASCII(fieldName) + `"` },
	},
	{Name: "int", Type: types.Typ[types.Int], Sample: func(string) string { return "42" }},
	{Name: "bool", Type: types.Typ[types.Bool], Sample: func(string) string { return "true" }},
	{Name: "float64", Type: types.Typ[types.Float64], Sample: func(string) string { return "3.14" }},

	// Signed integer family — sized variants for constraints that
	// require a specific width (e.g. `~int64` in audit-log offsets).
	{Name: "int8", Type: types.Typ[types.Int8], Sample: func(string) string { return "8" }},
	{Name: "int16", Type: types.Typ[types.Int16], Sample: func(string) string { return "16" }},
	{Name: "int32", Type: types.Typ[types.Int32], Sample: func(string) string { return "32" }},
	{Name: "int64", Type: types.Typ[types.Int64], Sample: func(string) string { return "64" }},

	// Unsigned integer family.
	{Name: "uint", Type: types.Typ[types.Uint], Sample: func(string) string { return "42" }},
	{Name: "uint8", Type: types.Typ[types.Uint8], Sample: func(string) string { return "8" }},
	{Name: "uint16", Type: types.Typ[types.Uint16], Sample: func(string) string { return "16" }},
	{Name: "uint32", Type: types.Typ[types.Uint32], Sample: func(string) string { return "32" }},
	{Name: "uint64", Type: types.Typ[types.Uint64], Sample: func(string) string { return "64" }},
	{Name: "uintptr", Type: types.Typ[types.Uintptr], Sample: func(string) string { return "0" }},

	// The remaining numeric kinds.
	{Name: "float32", Type: types.Typ[types.Float32], Sample: func(string) string { return "1.5" }},
	{Name: "complex64", Type: types.Typ[types.Complex64], Sample: func(string) string { return "complex(1, 2)" }},
	{Name: "complex128", Type: types.Typ[types.Complex128], Sample: func(string) string { return "complex(1, 2)" }},
}

// SelectConcreteType picks the first candidate (starting at
// startIdx, wrapping around) whose underlying type satisfies
// constraint. Returns nil when no candidate satisfies — callers
// decide the fallback.
//
// startIdx drives the round-robin so multi-parameter generics get
// distinct samples per position under unconstrained ("any") type
// parameters; narrow constraints (Numeric, comparable, …) fall
// through to the first satisfying candidate regardless.
func SelectConcreteType(constraint types.Type, candidates []ConcreteType, startIdx int) *ConcreteType {
	if len(candidates) == 0 {
		return nil
	}
	iface, _ := constraint.Underlying().(*types.Interface)
	if iface == nil {
		// Constraint isn't an interface (rare — would be a type
		// alias). Fall back to round-robin.
		i := ((startIdx % len(candidates)) + len(candidates)) % len(candidates)
		return &candidates[i]
	}
	n := len(candidates)
	for k := range n {
		i := ((startIdx+k)%n + n) % n
		if types.Satisfies(candidates[i].Type, iface) {
			return &candidates[i]
		}
	}
	return nil
}

// lowerASCII is a small helper so [DefaultConcreteTypes] doesn't
// pull strings into scan.go's import list. ASCII-only matches Go
// identifier rules for field names; non-ASCII would render the
// same way under strings.ToLower.
func lowerASCII(s string) string {
	b := make([]byte, len(s))
	for i := range len(s) {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
