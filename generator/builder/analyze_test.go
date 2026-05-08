// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/builder"
)

// runAnalyze loads `name` and analyzes the listed types. Most
// builder analyze tests follow this shape — keeping the helper
// avoids repeating the loadFixture / DefaultConfig dance.
func runAnalyze(t *testing.T, fixture string, args []string, opts generator.Options) *builder.Data {
	t.Helper()
	if opts.Output == "" {
		opts.Output = fixture + "test/builders.gen.go"
	}
	data, err := builder.Analyze(loadFixture(t, fixture), args, generator.DefaultConfig(), opts)
	testkit.NoError(t, err, "Analyze")
	return data
}

func TestAnalyze(t *testing.T) {
	t.Parallel()

	t.Run("populates one StructData per requested type", func(t *testing.T) {
		t.Parallel()
		data := runAnalyze(t, "structs", []string{"Item", "Order"}, generator.Options{})
		testkit.Len(t, data.Structs, 2, "two structs requested")
	})

	t.Run("Item field shapes detected and rendered", func(t *testing.T) {
		t.Parallel()
		data := runAnalyze(t, "structs", []string{"Item"}, generator.Options{})
		s := data.Structs[0]
		testkit.Equal(t, s.Name, "Item", "Name")
		testkit.Equal(t, s.BuilderName, "ItemBuilder", "BuilderName")

		byName := make(map[string]builder.FieldData, len(s.Fields))
		for _, f := range s.Fields {
			byName[f.Name] = f
		}

		// Scalars across kinds — string + numeric variants set
		// IsBasicComparable; time.Time stays uncomparable (struct).
		testkit.True(t, byName["ID"].IsBasicComparable, "string is basic-comparable")
		testkit.True(t, byName["Count"].IsBasicComparable, "int is basic-comparable")
		testkit.True(t, byName["Active"].IsBasicComparable, "bool is basic-comparable")
		testkit.True(t, byName["Ratio"].IsBasicComparable, "float64 is basic-comparable")
		testkit.True(t, byName["Created"].IsStruct, "time.Time → IsStruct")
		testkit.False(t, byName["Created"].IsBasicComparable, "time.Time not basic")

		// Slice non-byte vs []byte distinction.
		testkit.True(t, byName["Tags"].IsSlice, "[]string → IsSlice")
		testkit.Equal(t, byName["Tags"].ElemTypeStr, "string", "ElemTypeStr")
		testkit.True(t, byName["Codes"].IsSlice, "[]int → IsSlice")
		testkit.True(t, byName["Data"].IsBytes, "[]byte → IsBytes (not IsSlice)")
		testkit.False(t, byName["Data"].IsSlice, "[]byte must NOT also be IsSlice")

		// Maps: basic and non-string-key.
		testkit.True(t, byName["Metadata"].IsMap, "map is IsMap")
		testkit.Equal(t, byName["Metadata"].MapKeyTypeStr, "string", "string key")
		testkit.True(t, byName["Counters"].IsMap, "non-string-key map is IsMap")
		testkit.Equal(t, byName["Counters"].MapKeyTypeStr, "int", "int key")

		// Pointer.
		testkit.True(t, byName["Owner"].IsPointer, "*string is IsPointer")

		// Unexported field MUST NOT appear.
		_, hidden := byName["hidden"]
		testkit.False(t, hidden, "unexported `hidden` field must be filtered out")
	})

	t.Run("fields are sorted alphabetically for diff-stable output", func(t *testing.T) {
		t.Parallel()
		data := runAnalyze(t, "structs", []string{"Item"}, generator.Options{})
		names := make([]string, len(data.Structs[0].Fields))
		for i, f := range data.Structs[0].Fields {
			names[i] = f.Name
		}
		// Source order in fields.go is ID, Count, Active, ... but
		// analyze must emit alphabetical order so a field rename
		// or reorder doesn't churn the generated diff.
		sortedExpected := []string{
			"Active", "Codes", "Count", "Counters",
			"Created", "Data", "ID", "Metadata", "Owner",
			"Ratio", "Tags",
		}
		testkit.Equal(t, names, sortedExpected, "alphabetical sort")
	})

	t.Run("Order embedded + nested + pointer fields render correctly", func(t *testing.T) {
		t.Parallel()
		data := runAnalyze(t, "structs", []string{"Order"}, generator.Options{})
		s := data.Structs[0]

		byName := make(map[string]builder.FieldData, len(s.Fields))
		for _, f := range s.Fields {
			byName[f.Name] = f
		}
		testkit.True(t, byName["Metadata"].IsStruct, "embedded Metadata is IsStruct")
		testkit.True(t, byName["Billing"].IsStruct, "Billing Address is IsStruct")
		testkit.True(t, byName["Customer"].IsPointer, "*Customer is IsPointer")
		testkit.True(t, byName["ID"].IsBasicComparable, "ID string drives FirstComparableField")
	})

	t.Run("Holder has no basic-comparable field — FirstComparableField nil", func(t *testing.T) {
		t.Parallel()
		data := runAnalyze(t, "structs", []string{"Holder"}, generator.Options{})
		testkit.True(t, data.Structs[0].FirstComparableField() == nil,
			"Holder's interface/func/chan fields aren't basic-comparable")
	})

	t.Run("HasUnexportedFields detects hidden field on Item", func(t *testing.T) {
		t.Parallel()
		data := runAnalyze(t, "structs", []string{"Item"}, generator.Options{})
		testkit.True(t, data.Structs[0].HasUnexportedFields, "Item has `hidden`")
	})

	t.Run("Order has only exported fields → HasUnexportedFields false", func(t *testing.T) {
		t.Parallel()
		data := runAnalyze(t, "structs", []string{"Order"}, generator.Options{})
		testkit.False(t, data.Structs[0].HasUnexportedFields, "Order is fully exported")
	})

	t.Run("convention defaults discovered in source package", func(t *testing.T) {
		t.Parallel()
		// Profile + ProfileDefaults both live in the defaults
		// source package — exercises the source-pkg branch of
		// resolveDefaultsFactory before the sibling-pkg fallback.
		data := runAnalyze(t, "defaults", []string{"Profile"}, generator.Options{})
		s := data.Structs[0]
		testkit.True(t, s.HasDefaults, "factory found in source pkg")
		testkit.False(t, s.DefaultsFromOutput, "factory is NOT in output pkg")
		// Output in defaultstest, factory in defaults → call must
		// be source-qualified.
		testkit.Equal(t, s.DefaultsFunc, "defaults.ProfileDefaults",
			"source-qualified factory call")
	})

	t.Run("convention defaults discovered in sibling test package", func(t *testing.T) {
		t.Parallel()
		// Request lives in defaults; RequestDefaults() lives in
		// the sibling defaultstest package the generator emits
		// into. Analyze must find it via tryLoadOutputPackage.
		// WorkDir resolves Output relative to the testdata path:
		// "../testdata/defaults" + "defaultstest/" loads the
		// sibling pkg.
		data := runAnalyze(t, "defaults", []string{"Request"}, generator.Options{
			Output:  "defaultstest/convention.gen.go",
			WorkDir: "../testdata/defaults",
		})
		s := data.Structs[0]
		testkit.True(t, s.HasDefaults, "factory found in sibling pkg")
		testkit.True(t, s.DefaultsFromOutput, "found via output-pkg branch")
		testkit.Equal(t, s.DefaultsFunc, "RequestDefaults", "bare factory name")
		testkit.False(t, s.HasFieldDefaults,
			"factory wins over field directives")
	})

	t.Run("tryLoadOutputPackage returns nil when output dir matches source", func(t *testing.T) {
		t.Parallel()
		// Output in "." (same directory) → tryLoadOutputPackage
		// short-circuits to nil. resolveDefaultsFactory then falls
		// through with no factory and no error.
		data := runAnalyze(t, "defaults", []string{"Config"}, generator.Options{
			Output:  "config.gen.go",
			WorkDir: "../testdata/defaults",
		})
		testkit.False(t, data.Structs[0].HasDefaults,
			"no factory; output dir collapsed to source dir")
	})

	t.Run("tryLoadOutputPackage returns nil when output package fails to load", func(t *testing.T) {
		t.Parallel()
		// Non-existent sibling pkg → loader.Load fails → nil.
		// Should not surface as an Analyze error; just no factory.
		data := runAnalyze(t, "defaults", []string{"Config"}, generator.Options{
			Output:  "nonexistent/builders.gen.go",
			WorkDir: "../testdata/defaults",
		})
		testkit.False(t, data.Structs[0].HasDefaults, "load failure → no factory")
	})

	t.Run("field directives populate per-field defaults", func(t *testing.T) {
		t.Parallel()
		data := runAnalyze(t, "defaults", []string{"Config"}, generator.Options{})
		s := data.Structs[0]
		testkit.True(t, s.HasFieldDefaults, "Config has //testkit:default directives")
		testkit.False(t, s.HasDefaults, "no convention factory for Config")

		byName := make(map[string]builder.FieldData, len(s.Fields))
		for _, f := range s.Fields {
			byName[f.Name] = f
		}
		testkit.Equal(t, byName["Host"].DefaultValue, `"localhost"`, "Host default")
		testkit.Equal(t, byName["Port"].DefaultValue, "8080", "Port default")
		testkit.Equal(t, byName["Verbose"].DefaultValue, "true", "Verbose default")
		testkit.Equal(t, byName["Name"].DefaultValue, "", "Name has no directive")
	})

	t.Run("generic Container[T any] gets string concrete instantiation", func(t *testing.T) {
		t.Parallel()
		data := runAnalyze(t, "generics", []string{"Container"}, generator.Options{})
		s := data.Structs[0]
		testkit.True(t, s.IsGeneric, "Container is generic")
		testkit.Equal(t, s.TypeParamDecl, "[T any]", "TypeParamDecl")
		testkit.Equal(t, s.TypeParamArgs, "[T]", "TypeParamArgs")
		testkit.Equal(t, s.TestTypeArgs, "[string]",
			"any-constrained T → string from DefaultConcreteTypes[0]")
	})

	t.Run("generic Pair[A, B any] cycles concrete types per position", func(t *testing.T) {
		t.Parallel()
		data := runAnalyze(t, "generics", []string{"Pair"}, generator.Options{})
		testkit.Equal(t, data.Structs[0].TestTypeArgs, "[string, int]",
			"position 0 → string, position 1 → int")
	})

	t.Run("Numeric-constrained Stat[T] picks an int satisfying the union", func(t *testing.T) {
		t.Parallel()
		data := runAnalyze(t, "generics", []string{"Stat"}, generator.Options{})
		testkit.Equal(t, data.Structs[0].TestTypeArgs, "[int]",
			"Numeric ~int|~int64|~float64 — string is rejected, int satisfies")
	})

	t.Run("comparable-constrained Lookup[K comparable, V any] satisfies both", func(t *testing.T) {
		t.Parallel()
		data := runAnalyze(t, "generics", []string{"Lookup"}, generator.Options{})
		// position 0 (K comparable) → string satisfies; position 1
		// (V any) → int from rotation.
		testkit.Equal(t, data.Structs[0].TestTypeArgs, "[string, int]",
			"both constraints accept string + int")
	})

	t.Run("missing struct surfaces a hard error", func(t *testing.T) {
		t.Parallel()
		_, err := builder.Analyze(loadFixture(t, "structs"), []string{"DoesNotExist"},
			generator.DefaultConfig(), generator.Options{Output: "x.gen.go"})
		testkit.True(t, err != nil, "missing struct errors")
	})

	t.Run("empty args list surfaces a hard error", func(t *testing.T) {
		t.Parallel()
		_, err := builder.Analyze(loadFixture(t, "structs"), nil,
			generator.DefaultConfig(), generator.Options{Output: "x.gen.go"})
		testkit.True(t, err != nil, "no types specified")
	})
}
