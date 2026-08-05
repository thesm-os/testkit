// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package domhint

import "reflect"

// RequiresHint reports whether typ is non-reflection-generatable
// and therefore needs a domhint registration to produce values
// during model-based testing.
//
// The heuristic is conservative: types that rapid.Make can't
// synthesize cleanly return true. The rules:
//
//   - Func, Chan, UnsafePointer kinds always require a hint —
//     rapid has no way to invent a sensible value.
//   - Interface kinds require a hint — rapid can't pick an impl.
//   - Structs with one or more unexported fields require a hint —
//     reflection can't set them from outside the defining package.
//   - Pointer, Slice, Array, Map types delegate to their element
//     type: a slice of a hint-required element needs a hint;
//     a slice of int does not.
//   - All other kinds (Bool, Int*, Uint*, Float*, String, Struct
//     with only exported fields) are reflection-generatable.
//
// Used by the model generator's analyze step to error early when
// an opaque parameter lacks a //testkit:domain-gen directive.
func RequiresHint(typ reflect.Type) bool {
	if typ == nil {
		return false
	}
	switch typ.Kind() {
	case reflect.Func, reflect.Chan, reflect.UnsafePointer, reflect.Interface:
		return true
	case reflect.Pointer, reflect.Slice, reflect.Array:
		return RequiresHint(typ.Elem())
	case reflect.Map:
		return RequiresHint(typ.Key()) || RequiresHint(typ.Elem())
	case reflect.Struct:
		return hasUnexportedField(typ)
	default:
		return false
	}
}

// Resolve classifies a parameter type against the registry and
// returns one of three outcomes for the codegen path:
//
//   - hint != nil: a registered Hint is available; the generator
//     emits the `<Iface>ModelWith<Type>Gen(gen)` option.
//   - hint == nil && needsHint: the type requires a hint but none
//     is registered — codegen should error with directive
//     guidance.
//   - hint == nil && !needsHint: the type is reflectively
//     generatable; codegen falls back to `rapid.Make[T]`.
//
// The returned name is the consumer-facing display name when a
// hint is present; the type's String() otherwise.
func Resolve(r *Registry, typ reflect.Type) (hint any, name string, needsHint bool) {
	if typ == nil {
		return nil, "", false
	}
	h, n, ok := LookupByType(r, typ)
	if ok {
		return h, n, true
	}
	return nil, typ.String(), RequiresHint(typ)
}

// hasUnexportedField reports whether typ (which must be a Struct
// kind) has at least one unexported field. Caller verifies the
// kind precondition.
func hasUnexportedField(typ reflect.Type) bool {
	for f := range typ.Fields() {
		if !f.IsExported() {
			return true
		}
	}
	return false
}
