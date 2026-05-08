// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator

import (
	"fmt"
	"go/types"
	"strings"
)

// zeroNil is the literal "nil" — used as a sample value for any type
// whose only sensible non-zero is "no useful value here, default."
const zeroNil = "nil"

// SampleValueOf returns a non-zero Go literal for typ, suitable for
// use in generated test assertions. The value is deterministic — same
// type + same field name → same literal — and distinct from the zero
// value so tests can verify setters and round-trip semantics.
//
// fieldName is used to seed string-sample contents
// (`"test-<fieldname>"`) so tests can find specific samples in
// rendered Error() output.
//
// Used by sentinel (custom-error-type field samples) and by future
// generators that need synthetic test inputs.
func SampleValueOf(typ types.Type, fieldName string, tracker *ImportTracker) string {
	if named, ok := typ.(*types.Named); ok {
		qualifiedName := types.TypeString(typ, tracker.Qualifier())
		switch st := named.Underlying().(type) {
		case *types.Struct:
			return sampleStructLiteral(qualifiedName, st, tracker)
		case *types.Basic:
			inner := SampleValueOf(named.Underlying(), fieldName, tracker)
			return qualifiedName + "(" + inner + ")"
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
		return sampleStructLiteral(types.TypeString(typ, tracker.Qualifier()), u, tracker)
	case *types.Pointer:
		// Only produce &Type{...} for struct/named types — basic types
		// like *string can't use composite literal syntax.
		if st, isStruct := u.Elem().Underlying().(*types.Struct); isStruct {
			elemStr := types.TypeString(u.Elem(), tracker.Qualifier())
			return "&" + sampleStructLiteral(elemStr, st, tracker)
		}
		return zeroNil
	case *types.Signature, *types.Chan, *types.Interface:
		return zeroNil
	}
	return zeroNil
}

// SampleBasicValue returns a non-zero Go literal for a basic type.
// Strings render as `"test-<fieldname>"` so that the seeded value is
// recognizable when it appears in rendered output (e.g. an Error()
// string concatenation).
func SampleBasicValue(b *types.Basic, fieldName string) string {
	switch {
	case b.Info()&types.IsString != 0:
		return fmt.Sprintf(`"test-%s"`, strings.ToLower(fieldName))
	case b.Info()&types.IsBoolean != 0:
		return "true"
	case b.Info()&types.IsFloat != 0:
		return "3.14"
	case b.Info()&types.IsInteger != 0:
		return "42"
	default:
		return "0"
	}
}

// sampleStructLiteral produces a non-zero struct literal by populating
// the first exported basic-typed field with a sample value. When the
// struct has no suitable exported fields, falls back to the zero
// composite literal.
func sampleStructLiteral(qualifiedName string, st *types.Struct, tracker *ImportTracker) string {
	for f := range st.Fields() {
		if !f.Exported() {
			continue
		}
		if _, isBasic := f.Type().Underlying().(*types.Basic); !isBasic {
			continue
		}
		sample := SampleValueOf(f.Type(), f.Name(), tracker)
		return qualifiedName + "{" + f.Name() + ": " + sample + "}"
	}
	return qualifiedName + "{}"
}
