// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator

import (
	"fmt"
	"go/types"
	"strconv"
	"strings"
)

// zeroNil is the literal "nil" — used as a sample value for any type
// whose only sensible non-zero is "no useful value here, default."
const zeroNil = "nil"

// SampleForConcreteType returns a sample literal for a rendered
// concrete type name. Walks [DefaultConcreteTypes] for an exact
// name match and returns that candidate's Sample(fieldName).
// Falls through to a typed-zero-literal expression for unknown
// names (slice / map / unfamiliar type) — slices unwrap to
// `<sliceTypeStr>{<sample-of-elem>}`, others to `<typeName>{}`.
//
// Used by generators that resolve type-param names to concrete
// types (via [SubstituteTypeParams]) and then need a sample
// literal in the test view.
func SampleForConcreteType(typeName, fieldName string) string {
	for _, c := range DefaultConcreteTypes {
		if c.Name == typeName {
			return c.Sample(fieldName)
		}
	}
	if elemType, ok := strings.CutPrefix(typeName, "[]"); ok {
		return typeName + "{" + SampleForConcreteType(elemType, fieldName) + "}"
	}
	return typeName + "{}"
}

// ZeroParamExprs returns one zero-value Go expression per parameter,
// with [context.Context] replaced by `t.Context()` so generated
// auto-tests can call the stub directly:
//
//	sig: (ctx context.Context, key string)
//	    → []string{"t.Context()", `""`}
//	sig: (item Item, n int)
//	    → []string{"Item{}", "0"}
//	sig: (ctx context.Context, ids ...string)
//	    → []string{"t.Context()"} (variadic last param drops, since `f()` is a valid zero call)
//
// Used by stub auto-tests + future suite/bench test templates that
// need the smallest valid call expression for a method.
func ZeroParamExprs(sig *types.Signature, t *ImportTracker) []string {
	params := sig.Params()
	n := params.Len()
	if sig.Variadic() && n > 0 {
		n--
	}
	out := make([]string, 0, n)
	for i := range n {
		p := params.At(i)
		if i == 0 && IsContextType(p.Type()) {
			out = append(out, "t.Context()")
			continue
		}
		out = append(out, ZeroValueOf(p.Type(), t))
	}
	return out
}

// SampleParamExprs is the sample-value counterpart to
// [ZeroParamExprs] — fills non-ctx params with [SampleValueOf]
// rather than zero. The first (ctx) param still resolves to
// `t.Context()`.
//
//	sig: (ctx context.Context, key string)
//	    → []string{"t.Context()", `"test-key"`}
//
// Used by auto-tests verifying parameter recording (the values
// land in the Call struct's params slice).
func SampleParamExprs(sig *types.Signature, t *ImportTracker) []string {
	params := sig.Params()
	n := params.Len()
	if sig.Variadic() && n > 0 {
		n--
	}
	out := make([]string, 0, n)
	for i := range n {
		p := params.At(i)
		if i == 0 && IsContextType(p.Type()) {
			out = append(out, "t.Context()")
			continue
		}
		name := p.Name()
		if name == "" {
			name = ParamName(i)
		}
		out = append(out, SampleValueOf(p.Type(), name, t))
	}
	return out
}

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
	if _, ok := typ.(*types.TypeParam); ok {
		// Type parameters resolve to a concrete type only at the
		// instantiation site. Render the universal zero idiom
		// `*new(T)` so consumers' SubstituteTypeParams pass can
		// rewrite "T" → concrete and produce a valid (zero) value
		// for the resolved type. Sampling a non-zero literal is
		// impossible without the concrete map; tests that need a
		// non-zero value across generics should sample post-
		// substitution via [SampleForConcreteType].
		return "*new(" + TypeStr(typ, tracker) + ")"
	}
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
		// Element seed strips a single trailing 's' from a plural
		// param name so a `keys ...string` parameter samples as
		// `[]string{"test-key"}` rather than `[]string{"test-keys"}`,
		// keeping the rendered literal consistent with the
		// `test-key` convention used for non-variadic key slots.
		// Conservative: only strips when the result has at least
		// three characters and the name actually ends in 's'.
		elemName := fieldName
		if len(fieldName) > 3 && strings.HasSuffix(fieldName, "s") {
			elemName = strings.TrimSuffix(fieldName, "s")
		}
		sample := SampleValueOf(u.Elem(), elemName, tracker)
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
// Strings render as `"test-<fieldname>"` so the seeded value is
// recognizable when it appears in rendered output (e.g. an Error()
// string concatenation). Integers and floats derive from the
// fieldName's trailing digit when present so multi-slot signatures
// (MultiAggregator's two ints, MultiReader's two values) sample
// distinct values rather than colliding on the same default — a
// collision would prevent contract assertions from catching a
// tuple-swap bug. fieldName "Result0" → 42, "Result1" → 43, etc.;
// names without a trailing digit fall back to the default literal.
func SampleBasicValue(b *types.Basic, fieldName string) string {
	switch {
	case b.Info()&types.IsString != 0:
		return fmt.Sprintf(`"test-%s"`, strings.ToLower(fieldName))
	case b.Info()&types.IsBoolean != 0:
		return "true"
	case b.Info()&types.IsFloat != 0:
		return floatSampleFor(fieldName)
	case b.Info()&types.IsInteger != 0:
		return intSampleFor(fieldName)
	default:
		return "0"
	}
}

// intSampleFor returns "42" plus the trailing-digit offset of
// fieldName so multi-slot signatures sample distinct integers.
func intSampleFor(fieldName string) string {
	base := 42
	if n := len(fieldName); n > 0 {
		if c := fieldName[n-1]; c >= '0' && c <= '9' {
			base += int(c - '0')
		}
	}
	return strconv.Itoa(base)
}

// floatSampleFor returns "3.14" plus the trailing-digit offset of
// fieldName as a hundredths increment so multi-slot signatures
// sample distinct floats.
func floatSampleFor(fieldName string) string {
	base := 3.14
	if n := len(fieldName); n > 0 {
		if c := fieldName[n-1]; c >= '0' && c <= '9' {
			base += float64(c-'0') / 100
		}
	}
	return fmt.Sprintf("%g", base)
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
