// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"bytes"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/builder"
)

// renderPartial parses every embedded template and executes the
// named one against data, returning the raw output. Wrapping the
// boilerplate keeps each subtest focused on "this partial emits
// these markers".
func renderPartial(t *testing.T, name string, data any) string {
	t.Helper()
	tmpl, err := generator.NewTemplateSet().ParseFS(builder.TemplateFS(), "templates/*.tmpl")
	testkit.NoError(t, err, "ParseFS")
	var buf bytes.Buffer
	testkit.NoError(t, tmpl.ExecuteTemplate(&buf, name, data), "ExecuteTemplate "+name)
	return buf.String()
}

// sampleData builds a builder.Data populated to exercise every
// partial branch — a non-generic struct with rich field shapes
// plus a generic one with concrete instantiation.
func sampleData() *builder.Data {
	return &builder.Data{
		PackageName: "structstest",
		Structs: []builder.StructData{
			{
				Name:                "Item",
				BuilderName:         "ItemBuilder",
				QualifiedType:       "structs.Item",
				HasUnexportedFields: true,
				Fields: []builder.FieldData{
					{Name: "ID", TypeStr: "string", SampleValue: `"test-id"`, IsBasicComparable: true},
					{
						Name: "Tags", TypeStr: "[]string", IsSlice: true,
						ElemTypeStr: "string", SampleValue: `[]string{"a", "b"}`,
					},
					{
						Name: "Metadata", TypeStr: "map[string]string", IsMap: true,
						MapKeyTypeStr: "string", MapValTypeStr: "string",
						MapKeySample: `"k"`, MapValSample: `"v"`,
						SampleValue: `map[string]string{"k": "v"}`,
					},
					{Name: "Data", TypeStr: "[]byte", IsBytes: true, SampleValue: `[]byte("data")`},
				},
			},
			{
				Name:              "Container",
				BuilderName:       "ContainerBuilder",
				QualifiedType:     "generics.Container[T]",
				IsGeneric:         true,
				TypeParamDecl:     "[T any]",
				TypeParamArgs:     "[T]",
				TestTypeArgs:      "[string]",
				TestQualifiedType: "generics.Container[string]",
				Fields: []builder.FieldData{
					{
						Name: "Label", TypeStr: "string", SampleValue: `"test-label"`,
						IsBasicComparable: true,
						TestTypeStr:       "string", TestSample: `"test-label"`,
					},
				},
			},
		},
	}
}

// typeDict wraps one StructData with the enclosing Data so the
// "builder-test-type" partial sees the dict shape its template
// expects.
func typeDict(local *builder.Data, idx int) map[string]any {
	return map[string]any{
		"Type":  local.Structs[idx],
		"Local": local,
	}
}

func TestTemplateFS(t *testing.T) {
	t.Parallel()

	t.Run("embeds the five expected partials", func(t *testing.T) {
		t.Parallel()
		entries, err := builder.TemplateFS().ReadDir("templates")
		testkit.NoError(t, err, "ReadDir")
		want := map[string]bool{
			"builder.go.tmpl":           false,
			"builder_test.go.tmpl":      false,
			"header.go.tmpl":            false,
			"builder_type.go.tmpl":      false,
			"builder_test_type.go.tmpl": false,
		}
		for _, e := range entries {
			if _, ok := want[e.Name()]; ok {
				want[e.Name()] = true
			}
		}
		for name, found := range want {
			testkit.True(t, found, "missing template: "+name)
		}
	})
}

func TestPartial_Header(t *testing.T) {
	t.Parallel()

	t.Run("emits package + file-level doc", func(t *testing.T) {
		t.Parallel()
		out := renderPartial(t, "header", sampleData())
		testkit.Assert(t, out).
			Contains("package structstest", "package decl").
			Contains("// This file holds fluent builders", "doc lead-in").
			Contains("New<Type>()", "contract bullet present")
	})

	t.Run("import block omitted when no imports", func(t *testing.T) {
		t.Parallel()
		out := renderPartial(t, "header", sampleData())
		testkit.Assert(t, out).
			NotContains("import (", "no import block when Imports is empty")
	})

	t.Run("Directives consumed block surfaces only when set", func(t *testing.T) {
		t.Parallel()
		without := renderPartial(t, "header", sampleData())
		testkit.Assert(t, without).
			NotContains("Directives consumed:", "no block without directives")

		with := *sampleData()
		with.Directives = []string{"//testkit:example"}
		testkit.Assert(t, renderPartial(t, "header", &with)).
			Contains("Directives consumed:", "block present").
			Contains("//testkit:example", "directive line")
	})
}

func TestPartial_BuilderType(t *testing.T) {
	t.Parallel()

	t.Run("non-generic Item emits per-field-shape setters", func(t *testing.T) {
		t.Parallel()
		out := renderPartial(t, "builder-type", sampleData().Structs[0])
		testkit.Assert(t, out).
			Contains("type ItemBuilder struct", "type decl").
			Contains("WithID(v string)", "scalar").
			Contains("WithTags(v ...string)", "slice variadic").
			Contains("AppendTags(", "slice append").
			Contains("WithMetadata(m map[string]string)", "map setter").
			Contains("WithMetadataEntry(k string, v string)", "map entry").
			Contains("WithData(v []byte)", "bytes").
			Contains("WithDataString(s string)", "bytes-string convenience").
			Contains("Mutate(fn", "Mutate").
			Contains("Clone()", "Clone").
			Contains("Build()", "Build")
	})

	t.Run("generic Container[T] threads type params through", func(t *testing.T) {
		t.Parallel()
		out := renderPartial(t, "builder-type", sampleData().Structs[1])
		testkit.Assert(t, out).
			Contains("type ContainerBuilder[T any] struct", "generic type decl").
			Contains("func NewContainer[T any]()", "generic constructor").
			Contains("func (b *ContainerBuilder[T])", "generic receiver")
	})
}

func TestPartial_BuilderTestType(t *testing.T) {
	t.Parallel()

	t.Run("non-generic emits zero-value round-trip + all setters when no unexported", func(t *testing.T) {
		t.Parallel()
		data := sampleData()
		// Item has unexported — round-trip must be SUPPRESSED.
		out := renderPartial(t, "builder-test-type", typeDict(data, 0))
		testkit.Assert(t, out).
			Contains("func TestItemBuilder", "test fn").
			Contains(`"NewItem builds without panic"`, "smoke subtest").
			NotContains(`"NewItemFrom zero value round-trips"`,
				"round-trip suppressed when unexported fields exist")
	})

	t.Run("non-generic without unexported fields emits round-trip", func(t *testing.T) {
		t.Parallel()
		data := sampleData()
		// Flip the unexported flag off to exercise the "include
		// round-trip" path.
		data.Structs[0].HasUnexportedFields = false
		out := renderPartial(t, "builder-test-type", typeDict(data, 0))
		testkit.Assert(t, out).
			Contains(`"NewItemFrom zero value round-trips"`, "round-trip emitted")
	})

	t.Run("FirstComparableField gates Mutate and Clone subtests", func(t *testing.T) {
		t.Parallel()
		// Item has ID (basic-comparable) → Mutate / Clone subtests
		// are emitted asserting on ID.
		data := sampleData()
		data.Structs[0].HasUnexportedFields = false // simplify the check
		withCmp := renderPartial(t, "builder-test-type", typeDict(data, 0))
		testkit.Assert(t, withCmp).
			Contains(`"Clone forks independent scalar"`, "Clone subtest emitted").
			Contains(`"Mutate modifies value"`, "Mutate subtest emitted")

		// Strip ID to simulate "no comparable field" — Mutate /
		// Clone subtests must NOT appear.
		dataNoCmp := sampleData()
		dataNoCmp.Structs[0].HasUnexportedFields = false
		dataNoCmp.Structs[0].Fields = dataNoCmp.Structs[0].Fields[1:] // drop ID
		withoutCmp := renderPartial(t, "builder-test-type", typeDict(dataNoCmp, 0))
		testkit.Assert(t, withoutCmp).
			NotContains(`"Clone forks independent scalar"`,
				"Clone skipped without basic-comparable field").
			NotContains(`"Mutate modifies value"`,
				"Mutate skipped without basic-comparable field")
	})

	t.Run("generic test uses concrete instantiation", func(t *testing.T) {
		t.Parallel()
		out := renderPartial(t, "builder-test-type", typeDict(sampleData(), 1))
		testkit.Assert(t, out).
			Contains("func TestContainerBuilder", "test fn").
			Contains("Container[string]", "concrete instantiation").
			Contains("NewContainer[string]()", "concrete constructor call")
	})
}
