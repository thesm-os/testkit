// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import (
	"fmt"
	"go/types"
	"strings"
)

const (
	// ErrFieldName is the default name for error return fields.
	ErrFieldName = "Err"
	// ResultFieldName is the default name for non-error return fields.
	ResultFieldName = "Result"
)

// FieldData is a rendered struct field used by multiple generators
// (stub call types, builder fields, recording call types, suite specs).
type FieldData struct {
	FieldName string // "ID", "Name" — Go-initialism-aware
	TypeStr   string // "context.Context", "store.Item"
	ZeroValue string // `""`, `0`, `nil`, `Item{}`
	IsError   bool   // true if this is the error return
}

// BuildParamFields creates FieldData for each parameter in a function
// signature tuple. Parameter names are capitalized following Go
// initialism conventions.
func BuildParamFields(tuple *types.Tuple, tracker *ImportTracker) []FieldData {
	fields := make([]FieldData, 0, tuple.Len())
	for i := range tuple.Len() {
		v := tuple.At(i)
		name := v.Name()
		if name == "" {
			name = ParamName(i)
		}
		fields = append(fields, FieldData{
			FieldName: Title(name),
			TypeStr:   types.TypeString(v.Type(), tracker.Qualifier()),
			ZeroValue: ZeroValueOf(v.Type(), tracker),
		})
	}
	return fields
}

// BuildResultFields creates FieldData for each result in a function
// signature tuple. Names are derived from declared names when available,
// falling back to "Result"/"Result0"/"Err" conventions.
func BuildResultFields(tuple *types.Tuple, tracker *ImportTracker) []FieldData {
	fields := make([]FieldData, 0, tuple.Len())
	for i := range tuple.Len() {
		v := tuple.At(i)
		name := v.Name()
		isErr := IsErrorType(v.Type())
		if name == "" {
			if isErr {
				name = ErrFieldName
			} else if tuple.Len() == 1 || (tuple.Len() == 2 && IsErrorType(tuple.At(1).Type())) {
				name = ResultFieldName
			} else {
				name = ResultFieldName + string(rune('0'+i))
			}
		} else {
			name = Title(name)
		}
		fields = append(fields, FieldData{
			FieldName: name,
			TypeStr:   types.TypeString(v.Type(), tracker.Qualifier()),
			ZeroValue: ZeroValueOf(v.Type(), tracker),
			IsError:   isErr,
		})
	}
	return fields
}

// SampleValueOf returns a non-zero Go literal for a type, suitable for
// use in generated test assertions. The value is deterministic and
// distinct from the zero value so tests can verify setters work.
func SampleValueOf(typ types.Type, fieldName string, tracker *ImportTracker) string {
	if named, ok := typ.(*types.Named); ok {
		qualifiedName := types.TypeString(typ, tracker.Qualifier())
		switch named.Underlying().(type) {
		case *types.Struct:
			return qualifiedName + zeroSuffix
		case *types.Basic:
			innerSample := SampleValueOf(named.Underlying(), fieldName, tracker)
			return qualifiedName + "(" + innerSample + ")"
		default:
			return SampleValueOf(named.Underlying(), fieldName, tracker)
		}
	}

	switch u := typ.(type) {
	case *types.Basic:
		return SampleBasicValue(u, fieldName)
	case *types.Slice:
		elemStr := types.TypeString(u.Elem(), tracker.Qualifier())
		sample := SampleValueOf(u.Elem(), fieldName, tracker)
		return fmt.Sprintf("[]%s{%s}", elemStr, sample)
	case *types.Array:
		return types.TypeString(typ, tracker.Qualifier()) + "{1}"
	case *types.Map:
		keyStr := types.TypeString(u.Key(), tracker.Qualifier())
		valStr := types.TypeString(u.Elem(), tracker.Qualifier())
		keySample := SampleValueOf(u.Key(), fieldName, tracker)
		valSample := SampleValueOf(u.Elem(), fieldName, tracker)
		return fmt.Sprintf("map[%s]%s{%s: %s}", keyStr, valStr, keySample, valSample)
	case *types.Struct:
		return types.TypeString(typ, tracker.Qualifier()) + zeroSuffix
	case *types.Pointer:
		return zeroNil
	case *types.Signature, *types.Chan, *types.Interface:
		return zeroNil
	}
	return zeroNil
}

// SampleBasicValue returns a non-zero Go literal for a basic type.
func SampleBasicValue(b *types.Basic, fieldName string) string {
	switch {
	case b.Info()&types.IsString != 0:
		return fmt.Sprintf(`"test-%s"`, strings.ToLower(fieldName))
	case b.Info()&types.IsBoolean != 0:
		return "true"
	case b.Info()&types.IsInteger != 0:
		return "42"
	case b.Info()&types.IsFloat != 0:
		return "3.14"
	case b.Info()&types.IsUnsigned != 0:
		return "7"
	default:
		return "0"
	}
}

// HasUnexportedFields reports whether the type (or its named underlying
// struct) contains unexported fields, which prevents [cmp.Diff] comparison.
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
