// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import (
	"go/types"
	"strings"
)

const (
	contextPkgPath  = "context"
	contextTypName  = "Context"
	errorTypName    = "error"
	errorMethodName = "Error"
	iterPkgPath     = "iter"
	iterSeqName     = "Seq"
	iterSeq2Name    = "Seq2"

	zeroNil     = "nil"
	zeroFalse   = "false"
	zeroNumeric = "0"
	zeroString  = `""`
	zeroSuffix  = "{}"
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

// NumParams returns the number of parameters (excluding receiver).
func (m *MethodInfo) NumParams() int {
	return m.Signature.Params().Len()
}

// NumResults returns the number of result values.
func (m *MethodInfo) NumResults() int {
	return m.Signature.Results().Len()
}

// IsVariadic reports whether the last parameter is variadic.
func (m *MethodInfo) IsVariadic() bool {
	return m.Signature.Variadic()
}

// ParamList renders the parameter list as Go source using the given
// [ImportTracker] to qualify types.
//
//	"ctx context.Context, id string"
func (m *MethodInfo) ParamList(t *ImportTracker) string {
	return tupleString(m.Signature.Params(), t, m.Signature.Variadic())
}

// ParamNameList returns individual parameter names as a slice.
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

// ParamNames renders just the parameter names, comma-separated.
//
//	"ctx, id"
func (m *MethodInfo) ParamNames() string {
	return strings.Join(m.ParamNameList(), ", ")
}

// ParamNamesSpread renders parameter names for a forwarding call.
// For variadic methods, the last parameter is spread with "...".
//
//	"ctx, ids..." (variadic)
//	"ctx, id"    (non-variadic)
func (m *MethodInfo) ParamNamesSpread() string {
	names := m.ParamNameList()
	if m.Signature.Variadic() && len(names) > 0 {
		names[len(names)-1] += "..."
	}
	return strings.Join(names, ", ")
}

// ResultList renders the result type list as Go source.
//
//	"(Item, error)" or "error" for single result
func (m *MethodInfo) ResultList(t *ImportTracker) string {
	results := m.Signature.Results()
	if results.Len() == 0 {
		return ""
	}
	s := resultTupleString(results, t)
	if results.Len() > 1 {
		return "(" + s + ")"
	}
	return s
}

// CallForward renders a forwarding call expression. For variadic
// methods, the last parameter is spread with "...".
//
//	"recv.Get(ctx, id)"
//	"recv.Find(ctx, ids...)"
func (m *MethodInfo) CallForward(recv string) string {
	return recv + "." + m.Name + "(" + m.ParamNamesSpread() + ")"
}

// ZeroResults renders the zero values for all result types,
// comma-separated. Used for error return paths.
//
//	"Item{}, nil"
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

// FuncType renders the function type signature (without name).
//
//	"func(context.Context, model.PutRequest) error"
func (m *MethodInfo) FuncType(t *ImportTracker) string {
	var b strings.Builder
	b.WriteString("func(")
	params := m.Signature.Params()
	for i := range params.Len() {
		if i > 0 {
			b.WriteString(", ")
		}
		if m.Signature.Variadic() && i == params.Len()-1 {
			// Variadic: replace []T with ...T
			slice := params.At(i).Type().(*types.Slice)
			b.WriteString("...")
			b.WriteString(types.TypeString(slice.Elem(), t.Qualifier()))
		} else {
			b.WriteString(types.TypeString(params.At(i).Type(), t.Qualifier()))
		}
	}
	b.WriteString(")")
	results := m.Signature.Results()
	if results.Len() > 0 {
		b.WriteString(" ")
		b.WriteString(m.ResultList(t))
	}
	return b.String()
}

// TypeParamDecl renders the type parameter declaration for a generic
// interface or struct.
//
//	"[K comparable, V any]"
func (i *InterfaceInfo) TypeParamDecl(t *ImportTracker) string {
	return typeParamDecl(i.TypeParams, t)
}

// TypeParamArgs renders type parameter names for instantiation.
//
//	"[K, V]"
func (i *InterfaceInfo) TypeParamArgs() string {
	return typeParamArgs(i.TypeParams)
}

// TypeParamDecl renders the type parameter declaration for a generic struct.
func (s *StructInfo) TypeParamDecl(t *ImportTracker) string {
	return typeParamDecl(s.TypeParams, t)
}

// TypeParamArgs renders type parameter names for instantiation.
func (s *StructInfo) TypeParamArgs() string {
	return typeParamArgs(s.TypeParams)
}

// --- helpers ---

// resultTupleString renders result types without names.
func resultTupleString(tuple *types.Tuple, t *ImportTracker) string {
	parts := make([]string, tuple.Len())
	for i := range tuple.Len() {
		parts[i] = types.TypeString(tuple.At(i).Type(), t.Qualifier())
	}
	return strings.Join(parts, ", ")
}

func tupleString(tuple *types.Tuple, t *ImportTracker, variadic bool) string {
	parts := make([]string, tuple.Len())
	for i := range tuple.Len() {
		v := tuple.At(i)
		name := v.Name()
		if name == "" {
			name = ParamName(i)
		}
		typeStr := types.TypeString(v.Type(), t.Qualifier())
		if variadic && i == tuple.Len()-1 {
			// Replace []T with ...T for variadic parameters.
			slice, ok := v.Type().(*types.Slice)
			if ok {
				typeStr = "..." + types.TypeString(slice.Elem(), t.Qualifier())
			}
		}
		parts[i] = name + " " + typeStr
	}
	return strings.Join(parts, ", ")
}

// ZeroValueOf returns the Go zero-value literal for a type.
func ZeroValueOf(typ types.Type, t *ImportTracker) string {
	switch u := typ.Underlying().(type) {
	case *types.Basic:
		switch {
		case u.Info()&types.IsBoolean != 0:
			return zeroFalse
		case u.Info()&types.IsNumeric != 0:
			return zeroNumeric
		case u.Info()&types.IsString != 0:
			return zeroString
		}
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan,
		*types.Signature, *types.Interface:
		return zeroNil
	case *types.Struct:
		return types.TypeString(typ, t.Qualifier()) + zeroSuffix
	case *types.Array:
		return types.TypeString(typ, t.Qualifier()) + zeroSuffix
	}
	// Fallback for named types.
	if _, ok := typ.(*types.Named); ok {
		if _, isIface := typ.Underlying().(*types.Interface); isIface {
			return zeroNil
		}
		return types.TypeString(typ, t.Qualifier()) + zeroSuffix
	}
	return zeroNil
}

// TypeStr renders a Go type as source code using the tracker's qualifier.
func TypeStr(typ types.Type, tracker *ImportTracker) string {
	return types.TypeString(typ, tracker.Qualifier())
}

// IsContextType reports whether typ is context.Context.
func IsContextType(typ types.Type) bool {
	named, ok := typ.(*types.Named)
	if !ok {
		return false
	}
	return named.Obj().Pkg() != nil &&
		named.Obj().Pkg().Path() == contextPkgPath &&
		named.Obj().Name() == contextTypName
}

// IsErrorType reports whether typ is the built-in error interface.
func IsErrorType(typ types.Type) bool {
	named, ok := typ.(*types.Named)
	if ok {
		return named.Obj().Pkg() == nil && named.Obj().Name() == errorTypName
	}
	// The error type is a built-in interface, not always *types.Named.
	iface, ok := typ.Underlying().(*types.Interface)
	if !ok || iface.NumMethods() != 1 {
		return false
	}
	m := iface.Method(0)
	return m.Name() == errorMethodName && m.Type().(*types.Signature).Results().Len() == 1
}

// IterSeqInfo holds the result of inspecting a return type for iter.Seq
// or iter.Seq2. Zero value means the type is not an iterator.
type IterSeqInfo struct {
	IsSeq     bool   // true if iter.Seq[V]
	IsSeq2    bool   // true if iter.Seq2[K, V]
	Seq2Error bool   // true if iter.Seq2[V, error] — the error-yielding pattern
	ElemType  string // qualified type string for V (Seq) or K (Seq2)
	ValType   string // qualified type string for V in Seq2 (empty for Seq)
}

// AnalyzeIterReturn inspects typ and returns [IterSeqInfo] if it is
// iter.Seq[V] or iter.Seq2[K, V]. Returns zero value for non-iterator types.
func AnalyzeIterReturn(typ types.Type, tracker *ImportTracker) IterSeqInfo {
	named, ok := typ.(*types.Named)
	if !ok {
		return IterSeqInfo{}
	}
	obj := named.Obj()
	if obj.Pkg() == nil || obj.Pkg().Path() != iterPkgPath {
		return IterSeqInfo{}
	}
	targs := named.TypeArgs()
	if targs == nil {
		return IterSeqInfo{}
	}
	switch obj.Name() {
	case iterSeqName:
		if targs.Len() != 1 {
			return IterSeqInfo{}
		}
		return IterSeqInfo{
			IsSeq:    true,
			ElemType: types.TypeString(targs.At(0), tracker.Qualifier()),
		}
	case iterSeq2Name:
		if targs.Len() != 2 {
			return IterSeqInfo{}
		}
		info := IterSeqInfo{
			IsSeq2:   true,
			ElemType: types.TypeString(targs.At(0), tracker.Qualifier()),
			ValType:  types.TypeString(targs.At(1), tracker.Qualifier()),
		}
		if IsErrorType(targs.At(1)) {
			info.Seq2Error = true
		}
		return info
	}
	return IterSeqInfo{}
}

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
