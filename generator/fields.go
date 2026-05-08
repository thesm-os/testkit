// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator

import (
	"fmt"
	"go/types"
)

// BuildParamFields builds [FieldData] entries for a function's parameter
// tuple. Context parameters are included when keepContext is true,
// skipped otherwise (callers like stub call-type generation typically
// want context separated).
//
// Field names are produced by [Title] from the parameter name, with
// initialisms promoted. Unnamed parameters get synthesized names
// (P0, P1, ...).
//
// Variadic semantics: the last parameter is rendered with the "[]T"
// element form. Callers that need the variadic form in code
// generation use ParamNames() with the spread operator.
func BuildParamFields(tup *types.Tuple, tracker *ImportTracker, variadic, keepContext bool) []FieldData {
	n := tup.Len()
	if n == 0 {
		return nil
	}
	out := make([]FieldData, 0, n)
	for i := range n {
		v := tup.At(i)
		t := v.Type()
		if !keepContext && IsContextType(t) {
			continue
		}
		name := v.Name()
		if name == "" {
			name = fmt.Sprintf("p%d", i)
		}
		// Variadic last: keep []T form for storage in call structs.
		out = append(out, FieldData{
			FieldName: Title(name),
			TypeStr:   TypeStr(t, tracker),
			ZeroValue: ZeroValueOf(t, tracker),
		})
	}
	_ = variadic // reserved for callers that need variadic flagging in FieldData
	return out
}

// BuildResultFields builds [FieldData] entries for a function's result
// tuple. The last result, if of type error, is marked with IsError=true.
//
// Field names: named results use their declared name (title-cased and
// initialism-promoted). Unnamed results use position-based names —
// "Result" for a single non-error return, "Result0"/"Result1"/... for
// multiple. The error result is named "Err" by convention.
//
//	Get(...) (Item, error)             → [{Result, "Item", IsError:false},
//	                                       {Err, "error", IsError:true}]
//	Stats(...) (int, int, string, err) → [{Result0,...}, {Result1,...},
//	                                       {Result2,...}, {Err, IsError:true}]
//	Swap(...) (old, new V, err error)  → [{Old,...}, {New,...}, {Err,...}]
func BuildResultFields(tup *types.Tuple, tracker *ImportTracker) []FieldData {
	n := tup.Len()
	if n == 0 {
		return nil
	}

	// Count unnamed non-error results so we know whether to use
	// "Result" or "ResultN".
	nonErrCount := 0
	hasErr := false
	for i := range n {
		v := tup.At(i)
		if IsErrorType(v.Type()) {
			hasErr = true
			continue
		}
		nonErrCount++
	}
	multipleNonErr := nonErrCount > 1

	out := make([]FieldData, 0, n)
	unnamedIdx := 0
	for i := range n {
		v := tup.At(i)
		t := v.Type()
		isErr := IsErrorType(t)

		var name string
		switch {
		case v.Name() != "":
			name = Title(v.Name())
		case isErr:
			name = "Err"
		case multipleNonErr:
			name = fmt.Sprintf("Result%d", unnamedIdx)
			unnamedIdx++
		default:
			name = "Result"
			unnamedIdx++
		}

		out = append(out, FieldData{
			FieldName: name,
			TypeStr:   TypeStr(t, tracker),
			ZeroValue: ZeroValueOf(t, tracker),
			IsError:   isErr,
		})
	}
	_ = hasErr
	return out
}

// HasUnexportedFields reports whether typ (or its named underlying
// struct) contains at least one unexported field. The builder
// generator uses this to suppress the New<Type>From zero-value
// round-trip subtest, since [cmp.Diff] can't compare across an
// unexported boundary without an opt-in option.
func HasUnexportedFields(typ types.Type) bool {
	if named, ok := typ.(*types.Named); ok {
		typ = named.Underlying()
	}
	strct, ok := typ.(*types.Struct)
	if !ok {
		return false
	}
	for field := range strct.Fields() {
		if !field.Exported() {
			return true
		}
	}
	return false
}
