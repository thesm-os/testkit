// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator

import (
	"fmt"
	"go/types"
	"strings"
)

// Standard package paths used in shape detection and rendering.
const (
	contextPkgPath = "context"
	contextTypName = "Context"
	errorTypName   = "error"
	iterPkgPath    = "iter"
	iterSeqName    = "Seq"
	iterSeq2Name   = "Seq2"
)

// HasContext reports whether the first parameter is context.Context.
func (m *MethodInfo) HasContext() bool {
	params := m.Signature.Params()
	if params.Len() == 0 {
		return false
	}
	return IsContextType(params.At(0).Type())
}

// ReturnsError reports whether the last result is the error interface.
func (m *MethodInfo) ReturnsError() bool {
	results := m.Signature.Results()
	if results.Len() == 0 {
		return false
	}
	return IsErrorType(results.At(results.Len() - 1).Type())
}

// NumParams returns the parameter count (excluding receiver).
func (m *MethodInfo) NumParams() int { return m.Signature.Params().Len() }

// NumResults returns the count of result values.
func (m *MethodInfo) NumResults() int { return m.Signature.Results().Len() }

// IsVariadic reports whether the last parameter is variadic.
func (m *MethodInfo) IsVariadic() bool { return m.Signature.Variadic() }

// ParamList renders the parameter list as Go source, qualified via t.
//
//	"ctx context.Context, id string"
//	"" for void parameter list.
func (m *MethodInfo) ParamList(t *ImportTracker) string {
	return tupleString(m.Signature.Params(), t, m.Signature.Variadic())
}

// ParamNameList returns parameter names in source order. Synthesizes
// names for unnamed parameters using [ParamName].
func (m *MethodInfo) ParamNameList() []string {
	params := m.Signature.Params()
	names := make([]string, params.Len())
	for i := range params.Len() {
		name := params.At(i).Name()
		if name == "" {
			name = ParamName(i)
		}
		names[i] = name
	}
	return names
}

// ParamNames returns the comma-separated parameter names. Variadic
// parameters get the "..." spread suffix.
//
//	"ctx, id"
//	"ctx, ids..."
func (m *MethodInfo) ParamNames() string {
	names := m.ParamNameList()
	if m.IsVariadic() && len(names) > 0 {
		names[len(names)-1] += "..."
	}
	return strings.Join(names, ", ")
}

// ResultList renders the result list as Go source, qualified via t.
//
//	"(Item, error)"
//	"error"
//	"" for void.
//
// Wraps multi-return tuples in parens to produce syntactically valid
// Go. Single unnamed results render bare; named results force parens
// because Go syntax requires them.
func (m *MethodInfo) ResultList(t *ImportTracker) string {
	results := m.Signature.Results()
	n := results.Len()
	if n == 0 {
		return ""
	}
	body := tupleString(results, t, false)
	if n == 1 && results.At(0).Name() == "" {
		return body
	}
	return "(" + body + ")"
}

// CallForward renders a forwarding call expression. Variadic methods
// spread the last parameter.
//
//	"recv.Get(ctx, id)"
//	"recv.Find(ctx, ids...)"
func (m *MethodInfo) CallForward(recv string) string {
	return recv + "." + m.Name + "(" + m.ParamNames() + ")"
}

// ZeroResults returns a comma-separated list of zero values for each
// result type, suitable for return statements in error/fault paths.
//
//	"Item{}, nil"
//	"nil"
//	"" for void
func (m *MethodInfo) ZeroResults(t *ImportTracker) string {
	results := m.Signature.Results()
	if results.Len() == 0 {
		return ""
	}
	parts := make([]string, results.Len())
	for i := range results.Len() {
		parts[i] = ZeroValueOf(results.At(i).Type(), t)
	}
	return strings.Join(parts, ", ")
}

// FuncType renders the method's function type without name, qualified
// via t. Used for option-function fields like:
//
//	"func(context.Context, string) (Item, error)"
//	"func(context.Context)" for void return
func (m *MethodInfo) FuncType(t *ImportTracker) string {
	params := tupleTypeString(m.Signature.Params(), t, m.Signature.Variadic())
	results := m.Signature.Results()
	rn := results.Len()
	if rn == 0 {
		return "func(" + params + ")"
	}
	if rn == 1 {
		return "func(" + params + ") " + TypeStr(results.At(0).Type(), t)
	}
	return "func(" + params + ") (" + tupleTypeString(results, t, false) + ")"
}

// ParamName synthesizes a parameter name when the source has none.
//
//	ParamName(0) → "p0"
//	ParamName(1) → "p1"
func ParamName(i int) string { return fmt.Sprintf("p%d", i) }

// IsContextType reports whether t is context.Context.
func IsContextType(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj != nil && obj.Pkg() != nil &&
		obj.Pkg().Path() == contextPkgPath && obj.Name() == contextTypName
}

// IsErrorType reports whether t is the builtin error interface.
func IsErrorType(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj != nil && obj.Name() == errorTypName && obj.Pkg() == nil
}

// AnalyzeIterReturn inspects t for iter.Seq[T] or iter.Seq2[T, U] and
// returns a populated [IterSeqInfo] when it matches. The returned info
// has zero-value IsSeq/IsSeq2 fields when t is neither.
func AnalyzeIterReturn(t types.Type, tracker *ImportTracker) IterSeqInfo {
	named, ok := t.(*types.Named)
	if !ok {
		return IterSeqInfo{}
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil || obj.Pkg().Path() != iterPkgPath {
		return IterSeqInfo{}
	}
	args := named.TypeArgs()
	switch obj.Name() {
	case iterSeqName:
		if args == nil || args.Len() != 1 {
			return IterSeqInfo{}
		}
		return IterSeqInfo{
			IsSeq:   true,
			ValType: TypeStr(args.At(0), tracker),
		}
	case iterSeq2Name:
		if args == nil || args.Len() != 2 {
			return IterSeqInfo{}
		}
		info := IterSeqInfo{
			IsSeq2:  true,
			ValType: TypeStr(args.At(0), tracker),
			ErrType: TypeStr(args.At(1), tracker),
		}
		info.Seq2Error = IsErrorType(args.At(1))
		return info
	}
	return IterSeqInfo{}
}

// TypeStr renders a single types.Type using tracker for qualification.
//
//	TypeStr(intType, t)        → "int"
//	TypeStr(itemType, t)       → "store.Item"
//	TypeStr(sliceItemType, t)  → "[]store.Item"
func TypeStr(t types.Type, tracker *ImportTracker) string {
	if tracker == nil {
		return types.TypeString(t, nil)
	}
	return types.TypeString(t, tracker.Qualifier())
}

// ZeroValueOf returns a Go expression that evaluates to the zero value
// of t. Picks the natural literal:
//
//	bool                          → "false"
//	intN/uintN/floatN/byte/rune   → "0"
//	string                        → `""`
//	pointer/slice/map/chan/func/interface → "nil"
//	struct                        → "<rendered type>{}"
//	array                         → "<rendered type>{}"
//	named type with one of the above → derived from underlying
//
// Type-parameter types render as `<T>(...)` zero (uses var-zero idiom).
func ZeroValueOf(t types.Type, tracker *ImportTracker) string {
	// Type parameter: use "var z T" idiom — but for inline expression we
	// render *(new(T)) which is always valid for any type.
	if _, ok := t.(*types.TypeParam); ok {
		return "*new(" + TypeStr(t, tracker) + ")"
	}

	switch u := t.Underlying().(type) {
	case *types.Basic:
		switch u.Kind() {
		case types.Bool:
			return "false"
		case types.String, types.UntypedString:
			return `""`
		case types.UntypedNil:
			return "nil"
		default:
			// All numeric kinds (int, uint, intN, uintN, floatN, complexN, byte, rune).
			return "0"
		}
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan, *types.Signature, *types.Interface:
		return "nil"
	case *types.Struct, *types.Array:
		return TypeStr(t, tracker) + "{}"
	}
	// Fallback — should be unreachable for well-typed Go.
	return TypeStr(t, tracker) + "{}"
}

// tupleString renders a function-parameter or function-result tuple as
// Go source with names. Variadic flag spreads the last parameter as
// "name ...T" (callers pass false for result tuples).
func tupleString(tup *types.Tuple, tracker *ImportTracker, variadic bool) string {
	n := tup.Len()
	if n == 0 {
		return ""
	}
	parts := make([]string, n)
	for i := range n {
		v := tup.At(i)
		name := v.Name()
		t := v.Type()
		typeStr := TypeStr(t, tracker)
		if variadic && i == n-1 {
			// Variadic: type is *types.Slice — strip "[]" prefix and add "...".
			if slice, ok := t.(*types.Slice); ok {
				typeStr = "..." + TypeStr(slice.Elem(), tracker)
			}
		}
		if name == "" {
			parts[i] = typeStr
		} else {
			parts[i] = name + " " + typeStr
		}
	}
	out := strings.Join(parts, ", ")
	if tup == nil {
		return out
	}
	// Single unnamed return type: render without parens.
	if n == 1 && tup.At(0).Name() == "" && !variadic {
		// Caller decides whether to wrap.
		return out
	}
	return out
}

// TypeParamDecl renders the type parameter declaration for a
// generic interface, e.g. `[K comparable, V any]`. Returns the
// empty string for non-generic interfaces. Used by stub/suite/bench
// when emitting generic interface types.
func (i *InterfaceInfo) TypeParamDecl(t *ImportTracker) string {
	return typeParamDecl(i.TypeParams, t)
}

// TypeParamArgs renders the type-parameter names for an
// instantiation, e.g. `[K, V]`. Returns the empty string for
// non-generic interfaces.
func (i *InterfaceInfo) TypeParamArgs() string {
	return typeParamArgs(i.TypeParams)
}

// TypeParamDecl renders the type parameter declaration for a
// generic struct. See [InterfaceInfo.TypeParamDecl].
func (s *StructInfo) TypeParamDecl(t *ImportTracker) string {
	return typeParamDecl(s.TypeParams, t)
}

// TypeParamArgs renders the type-parameter names for an
// instantiation. See [InterfaceInfo.TypeParamArgs].
func (s *StructInfo) TypeParamArgs() string {
	return typeParamArgs(s.TypeParams)
}

// typeParamDecl is the shared implementation: renders
// `[<name> <constraint>, ...]` from a TypeParamInfo slice. Empty
// slice → empty string.
func typeParamDecl(params []TypeParamInfo, t *ImportTracker) string {
	if len(params) == 0 {
		return ""
	}
	parts := make([]string, len(params))
	for i, tp := range params {
		parts[i] = tp.Name + " " + types.TypeString(tp.Constraint, t.Qualifier())
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// typeParamArgs is the shared implementation: renders
// `[<name>, ...]` from a TypeParamInfo slice. Empty slice → empty
// string.
func typeParamArgs(params []TypeParamInfo) string {
	if len(params) == 0 {
		return ""
	}
	names := make([]string, len(params))
	for i, tp := range params {
		names[i] = tp.Name
	}
	return "[" + strings.Join(names, ", ") + "]"
}

// QualifyType prefixes typeName with qualifier when non-empty:
//
//	QualifyType("store", "Item") → "store.Item"
//	QualifyType("",      "Item") → "Item"
//
// Used wherever templates must render a source-package-qualified
// type name from a separate qualifier and base name.
func QualifyType(qualifier, typeName string) string {
	if qualifier == "" {
		return typeName
	}
	return qualifier + "." + typeName
}

// tupleTypeString renders a tuple's types only, dropping names. Used
// for FuncType rendering where parameter names are not part of the
// type signature. The result is always a bare comma-separated list;
// callers wrap in parens themselves when their syntactic context
// requires it.
func tupleTypeString(tup *types.Tuple, tracker *ImportTracker, variadic bool) string {
	n := tup.Len()
	if n == 0 {
		return ""
	}
	parts := make([]string, n)
	for i := range n {
		v := tup.At(i)
		t := v.Type()
		typeStr := TypeStr(t, tracker)
		if variadic && i == n-1 {
			if slice, ok := t.(*types.Slice); ok {
				typeStr = "..." + TypeStr(slice.Elem(), tracker)
			}
		}
		parts[i] = typeStr
	}
	return strings.Join(parts, ", ")
}
