// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder

import "go.thesmos.sh/testkit/generator"

// Data is the top-level template input for one builder generation.
// Two output files are rendered from the same Data — the impl
// (`builder.go.tmpl`) and the test (`builder_test.go.tmpl`); both
// share PackageName / Imports / Structs but the test view also
// carries GenQualifier (the dotted prefix used to reference
// generated types from the sibling test package).
type Data struct {
	PackageName string

	// Imports is the resolved import set for the output file,
	// computed via [generator.ImportTracker]. Both impl and test
	// renders consume this list.
	Imports []generator.Import

	// Structs is one [StructData] per requested type, sorted by
	// declaration order in args (i.e. stable across runs).
	Structs []StructData

	// GenQualifier is the dotted prefix the test file uses to
	// reference symbols emitted into the impl package
	// ("storetest." or ""). Empty for internal-style tests where
	// the test lives in the same package as the impl.
	GenQualifier string

	// ImplImportPath is the import path of the impl file's
	// package, computed via [generator.BuildOutputCtx]. The
	// builder-test renderer's Transform hook uses it to compute
	// the test file's view via [generator.BuildTestFileInfo].
	ImplImportPath string

	// Directives are the //testkit: package-level directives
	// consumed by this generation, pre-rendered as source lines
	// for the header partial. Empty when none apply (builder
	// consumes none today; reserved for future).
	Directives []string
}

// HasContent reports whether the package has any structs worth
// generating builders for. Returns false only when args is empty,
// which would normally have errored earlier — included for
// [generator.SkippableData] symmetry with sentinel/enum.
func (d *Data) HasContent() bool {
	return len(d.Structs) > 0
}

// StructData holds one struct's full metadata for the builder
// generator. Pre-rendered fields (TypeParamDecl, TypeParamArgs,
// QualifiedType, etc.) keep the templates free of go/types-aware
// rendering.
type StructData struct {
	Name        string // "Item"
	BuilderName string // "ItemBuilder"

	// QualifiedType is the source-package-qualified type reference,
	// suffixed with type-args for generics. Examples:
	//   non-generic:        "structs.Item"
	//   generic Container:  "generics.Container[T]"
	//   generic Pair:       "generics.Pair[A, B]"
	QualifiedType string

	// Fields is one [FieldData] per exported field, sorted
	// alphabetically for diff-stable output. Unexported fields
	// are filtered out — they don't get setters.
	Fields []FieldData

	// HasUnexportedFields signals that the struct has at least one
	// unexported field. The generated New<Type>From assertion is
	// suppressed in this case (the zero-value test would compare
	// against a literal that excludes the unexported fields).
	HasUnexportedFields bool

	// HasDefaults is true when a `<Type>Defaults() <Type>` factory
	// exists in either the source or sibling test package. The
	// generated New<Type>() seeds via the factory.
	HasDefaults bool

	// DefaultsFunc is the call expression for the factory as seen
	// from the file currently being rendered. The impl renderer
	// receives an impl-side form ("ItemDefaults" when the factory
	// is in the same package as the impl, or "<src>.ItemDefaults"
	// when it lives in the source package); the test renderer's
	// Transform hook rewrites this to the test-side form
	// (prepending the test's GenQualifier when the factory is in
	// the output package). Empty when HasDefaults is false.
	DefaultsFunc string

	// DefaultsFromOutput is true when the factory was discovered in
	// the output package (sibling test pkg variant), false when in
	// the source package. The test-view Transform consults this to
	// pick whether to prepend GenQualifier.
	DefaultsFromOutput bool

	// HasFieldDefaults is true when any field carries a
	// //testkit:default directive. New<Type>() composes a literal
	// from those values (no factory call).
	HasFieldDefaults bool

	// Generic plumbing — empty for non-generic types.
	//
	//   TypeParamDecl       "[T any]" / "[K comparable, V any]"
	//   TypeParamArgs       "[T]"     / "[K, V]"
	//   IsGeneric           true when TypeParamDecl != ""
	//   TestTypeArgs        "[string]"  / "[string, int]" — concrete
	//                       instantiation used by generated tests.
	//   TestQualifiedType   "generics.Container[string]" — the test
	//                       file's local reference to the generic
	//                       struct with concrete types.
	TypeParamDecl     string
	TypeParamArgs     string
	IsGeneric         bool
	TestTypeArgs      string
	TestQualifiedType string
}

// FirstComparableField returns the first basic-comparable field
// (string / int* / uint* / bool / float* / rune / byte). Used by
// generated Mutate / Clone subtests to assert independence — those
// tests need a field whose non-zero sample is meaningfully different
// from the zero value.
//
// Returns nil when no field qualifies. Templates gate Mutate /
// Clone subtest emission on `{{with}}` (nil pointer is falsy in
// text/template), so the subtests are skipped cleanly for structs
// whose only fields are interfaces, funcs, channels, slices, maps,
// nested structs, or pointers — the kinds whose zero-vs-sample
// comparison can't drive a useful "different" assertion.
//
// The pointer return is load-bearing: returning a zero [FieldData]
// instead would render the subtests with empty field references
// because text/template's `{{with}}` doesn't treat a zero struct
// value as falsy.
func (s StructData) FirstComparableField() *FieldData {
	for i := range s.Fields {
		if s.Fields[i].IsBasicComparable {
			return &s.Fields[i]
		}
	}
	return nil
}

// FieldData is the rendered form of one exported struct field.
// Type-shape flags (IsSlice / IsMap / IsBytes / IsStruct /
// IsPointer) drive specialized setter emission; flags are mutually
// exclusive except IsPointer, which can co-occur with the others.
type FieldData struct {
	Name string

	// TypeStr is the field's Go-source type, qualified via the
	// package's [generator.ImportTracker]. For generic fields the
	// declared type-parameter names appear here; the test render
	// substitutes them via TestTypeStr.
	TypeStr string

	// SampleValue is a non-zero literal expression for the field's
	// type, suitable for the source-side With<Field> assertion.
	// String fields render as `"test-<lowername>"`; numeric fields
	// render plain digits; complex types render zero-or-typical
	// literal forms via [generator.SampleValueOf].
	SampleValue string

	// Slice plumbing.
	IsSlice     bool
	ElemTypeStr string // element type for `[]E` — drives variadic With + Append

	// Map plumbing.
	IsMap         bool
	MapKeyTypeStr string
	MapValTypeStr string
	MapKeySample  string
	MapValSample  string

	// IsBytes flags `[]byte` specifically — distinct from IsSlice
	// because bytes get a WithString convenience setter.
	IsBytes bool

	// Composition-shape flags. IsStruct covers both named and
	// anonymous struct fields; IsPointer covers any pointer type
	// (to scalar or struct).
	IsStruct  bool
	IsPointer bool

	// IsBasicComparable is true when the field's underlying type
	// is a built-in basic kind (string, int*, uint*, bool, float*,
	// rune, byte) — values with a meaningful non-zero literal
	// that `==` / `!=` operate on usefully. Drives
	// [StructData.FirstComparableField] so the generated Mutate /
	// Clone independence subtests pick a field whose mutation is
	// observable; interface, func, and chan fields are excluded
	// because their nil samples can't drive a "different"
	// assertion.
	IsBasicComparable bool

	// DefaultValue is the //testkit:default directive's argument
	// (verbatim, including quotes for strings). Empty when no
	// directive applies. Drives the New<Type>() field-defaults
	// composition and the matching test assertion.
	DefaultValue string

	// Generic test rendering. For non-generic fields these are
	// empty and templates fall back to TypeStr / SampleValue (and,
	// for map fields, MapKeySample / MapValSample).
	TestTypeStr      string // concrete type name for resolved type params
	TestSample       string // sample value matching TestTypeStr
	TestMapKeySample string // map-key sample using the resolved-K concrete type
	TestMapValSample string // map-val sample using the resolved-V concrete type
}

// EffectiveSample returns TestSample when set (generic fields with
// a resolved concrete instantiation) else SampleValue. Templates
// rendering test bodies must always use this — otherwise generic
// field tests would compare against samples typed in T, not the
// concrete instantiation.
func (f FieldData) EffectiveSample() string {
	if f.TestSample != "" {
		return f.TestSample
	}
	return f.SampleValue
}
