// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enum_test

import (
	"bytes"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/enum"
)

// renderPartial parses every embedded template and executes the
// named one against data, returning the raw output. Wrapping the
// boilerplate keeps each subtest focused on "this template emits
// these markers".
func renderPartial(t *testing.T, name string, data any) string {
	t.Helper()
	tmpl, err := generator.NewTemplateSet().ParseFS(enum.TemplateFS(), "templates/*.tmpl")
	testkit.NoError(t, err, "ParseFS")
	var buf bytes.Buffer
	testkit.NoError(t, tmpl.ExecuteTemplate(&buf, name, data), "ExecuteTemplate "+name)
	return buf.String()
}

// sampleData builds an enum.Data populated to exercise every partial
// branch. Two types — Status (full surface) and Priority (bare) —
// share one combined golden file so per-type wire-compat subtests
// assert their slice via [golden.AssertGoldenJSONField].
func sampleData() *enum.Data {
	return &enum.Data{
		PackageName: "basic_test",
		ImportPath:  "example.com/basic",
		Qualifier:   "basic.",
		GoldenFile:  "status.gen_wire.json",
		Enums: []enum.TypeData{
			{
				TypeName:         "Status",
				Qualifier:        "basic.",
				HasString:        true,
				HasParse:         true,
				ParseFunc:        "ParseStatus",
				HasMarshalText:   true,
				HasMarshalJSON:   true,
				HasMarshalBinary: true,
				MaxValue:         2,
				ZeroValueName:    "StatusPending",
				Values: []enum.Value{
					{Name: "StatusPending", ExpectedStr: "Pending", IntValue: 0},
					{Name: "StatusActive", ExpectedStr: "Active", IntValue: 1},
					{Name: "StatusClosed", ExpectedStr: "Closed", IntValue: 2},
				},
			},
			{
				TypeName:      "Priority",
				Qualifier:     "basic.",
				MaxValue:      2,
				ZeroValueName: "PriorityLow",
				Values: []enum.Value{
					{Name: "PriorityLow", ExpectedStr: "Low", IntValue: 0},
					{Name: "PriorityMedium", ExpectedStr: "Medium", IntValue: 1},
					{Name: "PriorityHigh", ExpectedStr: "High", IntValue: 2},
				},
			},
		},
	}
}

// typeDict wraps one TypeData with the enclosing Data so the
// "enum-type" partial sees the dict shape its template expects.
func typeDict(local *enum.Data, idx int) map[string]any {
	return map[string]any{
		"Type":  local.Enums[idx],
		"Local": local,
	}
}

func TestTemplateFS(t *testing.T) {
	t.Parallel()

	t.Run("embeds the three expected partials", func(t *testing.T) {
		t.Parallel()
		entries, err := enum.TemplateFS().ReadDir("templates")
		testkit.NoError(t, err, "ReadDir")
		want := map[string]bool{
			"enum.go.tmpl":      false,
			"header.go.tmpl":    false,
			"enum_type.go.tmpl": false,
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

	t.Run("emits package + canonical imports + file doc", func(t *testing.T) {
		t.Parallel()
		out := renderPartial(t, "header", sampleData())
		testkit.Assert(t, out).
			Contains("package basic_test", "package decl").
			Contains("// This file verifies", "file doc lead-in").
			Contains(`"encoding/json"`, "json always imported").
			Contains(`"testing"`, "testing import").
			Contains(`"go.thesmos.sh/testkit/golden"`, "golden import").
			Contains(`"example.com/basic"`, "source pkg import")
	})

	t.Run("conditional imports gated on rollups", func(t *testing.T) {
		t.Parallel()
		// All flags true: bytes + fmt imported.
		full := renderPartial(t, "header", sampleData())
		testkit.Assert(t, full).
			Contains(`"bytes"`, "bytes imported when binary").
			Contains(`"fmt"`, "fmt imported when stringer")

		// Bare data: no methods → no bytes, no fmt.
		bare := *sampleData()
		bare.Enums = []enum.TypeData{bare.Enums[1]} // Priority only
		testkit.Assert(t, renderPartial(t, "header", &bare)).
			NotContains(`"bytes"`, "no bytes when no binary").
			NotContains(`"fmt"`, "no fmt when no stringer")
	})

	t.Run("Directives consumed block surfaces only when set", func(t *testing.T) {
		t.Parallel()
		without := renderPartial(t, "header", sampleData())
		testkit.Assert(t, without).
			NotContains("Directives consumed:", "no block without directives")

		with := *sampleData()
		with.Directives = []string{"//testkit:wire-break-allowed"}
		testkit.Assert(t, renderPartial(t, "header", &with)).
			Contains("Directives consumed:", "block present").
			Contains("//testkit:wire-break-allowed", "directive line")
	})

	t.Run("source-pkg import omitted when same package", func(t *testing.T) {
		t.Parallel()
		samePkg := *sampleData()
		samePkg.ImportPath = ""
		testkit.Assert(t, renderPartial(t, "header", &samePkg)).
			NotContains("example.com/basic", "no extra import for same-pkg output")
	})
}

func TestPartial_EnumType(t *testing.T) {
	t.Parallel()

	t.Run("Status emits every conditional subtest plus wire compatibility", func(t *testing.T) {
		t.Parallel()
		out := renderPartial(t, "enum-type", typeDict(sampleData(), 0))
		testkit.Assert(t, out).
			Contains("func TestStatusEnum", "main test fn").
			Contains(`"exhaustive"`, "exhaustive subtest").
			Contains(`"all values are distinct"`, "distinct subtest").
			Contains(`"zero value is StatusPending"`, "zero-value subtest").
			Contains(`"stringer"`, "stringer subtest").
			Contains(`"out of range uses fallback format"`, "boundary subtest").
			Contains(`"parse round-trip"`, "parse subtest").
			Contains(`"marshal text round-trip"`, "text subtest").
			Contains(`"json round-trip"`, "json subtest").
			Contains(`"binary round-trip"`, "binary subtest").
			Contains("bytes.Equal", "binary determinism").
			Contains(`"wire compatibility"`, "wire-compat subtest").
			Contains(`golden.AssertGoldenJSONField`,
				"wire-compat uses the field-keyed assertion").
			Contains(`"status.gen_wire.json", "Status"`,
				"asserts the Status slice of the combined golden")
	})

	t.Run("Priority skips method-gated subtests but keeps wire compatibility", func(t *testing.T) {
		t.Parallel()
		out := renderPartial(t, "enum-type", typeDict(sampleData(), 1))
		testkit.Assert(t, out).
			Contains("func TestPriorityEnum", "main test fn").
			Contains(`"exhaustive"`, "exhaustive always").
			Contains(`"all values are distinct"`, "distinct always").
			Contains(`"wire compatibility"`, "wire-compat always present").
			Contains(`"status.gen_wire.json", "Priority"`,
				"asserts the Priority slice of the same combined golden").
			NotContains(`"stringer"`, "no stringer").
			NotContains(`"parse round-trip"`, "no parse").
			NotContains(`"marshal text"`, "no text").
			NotContains(`"json round-trip"`, "no json").
			NotContains(`"binary round-trip"`, "no binary")
	})

	t.Run("no top-level wire-compat function leaks across types", func(t *testing.T) {
		t.Parallel()
		// Wire compat lives inside Test<Type>Enum as a subtest, so
		// multiple types in one file cannot collide on a function name.
		out := renderPartial(t, "enum-type", typeDict(sampleData(), 0))
		testkit.Assert(t, out).
			NotContains("func TestEnumsWireCompat",
				"old top-level wire-compat must not return").
			NotContains("func TestStatusEnumWireCompat",
				"per-type top-level wire-compat must not return")
	})

	t.Run("ZeroValueName=\"\" suppresses the zero-value subtest", func(t *testing.T) {
		t.Parallel()
		data := sampleData()
		data.Enums[1].ZeroValueName = ""
		out := renderPartial(t, "enum-type", typeDict(data, 1))
		testkit.Assert(t, out).
			NotContains("zero value is", "no zero-value test when no const has value 0")
	})

	t.Run("function doc names the type", func(t *testing.T) {
		t.Parallel()
		out := renderPartial(t, "enum-type", typeDict(sampleData(), 0))
		testkit.Assert(t, out).
			Contains("// TestStatusEnum verifies", "function-level doc")
	})
}
