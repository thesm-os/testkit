// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enum_test

import (
	"slices"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/enum"
)

func TestGenerator(t *testing.T) {
	t.Parallel()

	t.Run("Name returns subcommand", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, (&enum.Generator{}).Name(), "enum", "Name")
	})

	t.Run("Generate emits one test file plus one combined golden", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&enum.Generator{}).Generate(pkg, []string{"Status", "Priority"},
			generator.DefaultConfig(), generator.Options{Output: "status.gen_test.go"})
		testkit.NoError(t, err, "Generate")
		testkit.Len(t, res.Files, 2, "test file + single combined golden")

		paths := make([]string, len(res.Files))
		for i, f := range res.Files {
			paths[i] = f.Path
		}
		testkit.Equal(t, paths[0], "status.gen_test.go", "test file first")
		testkit.True(t, hasPath(paths, "status.gen_wire.json"),
			"single combined golden alongside test file")
	})

	t.Run("test file documents what each subtest asserts", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&enum.Generator{}).Generate(pkg, []string{"Status"},
			generator.DefaultConfig(), generator.Options{Output: "status.gen_test.go"})
		testkit.NoError(t, err, "Generate")
		got := string(res.Files[0].Content)
		testkit.Assert(t, got).
			Contains("// This file verifies", "file-level doc summary").
			Contains("// TestStatusEnum verifies", "function-level doc").
			Contains(`// "exhaustive":`, "subtest comment").
			Contains(`// "all values are distinct":`, "subtest comment").
			Contains(`// "wire compatibility":`, "wire-compat subtest comment")
	})

	t.Run("Status path emits every conditional subtest plus wire compatibility", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&enum.Generator{}).Generate(pkg, []string{"Status"},
			generator.DefaultConfig(), generator.Options{Output: "status.gen_test.go"})
		testkit.NoError(t, err, "Generate")
		got := string(res.Files[0].Content)
		testkit.Assert(t, got).
			Contains("TestStatusEnum", "main test").
			Contains(`"stringer"`, "stringer subtest").
			Contains(`"out of range uses fallback format"`, "boundary subtest").
			Contains(`"parse round-trip"`, "parse subtest").
			Contains(`"marshal text round-trip"`, "text subtest").
			Contains(`"json round-trip"`, "json subtest").
			Contains(`"binary round-trip"`, "binary subtest").
			Contains(`"wire compatibility"`, "wire-compat subtest").
			Contains("bytes.Equal", "binary determinism check uses bytes.Equal").
			Contains("golden.AssertGoldenJSONField",
				"wire-compat uses field-keyed assertion").
			Contains(`"status.gen_wire.json", "Status"`,
				"asserts the Status slice of the combined golden")
	})

	t.Run("Priority path skips method-gated subtests but keeps wire compatibility", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&enum.Generator{}).Generate(pkg, []string{"Priority"},
			generator.DefaultConfig(), generator.Options{Output: "priority.gen_test.go"})
		testkit.NoError(t, err, "Generate")
		got := string(res.Files[0].Content)
		testkit.Assert(t, got).
			Contains("TestPriorityEnum", "main test").
			Contains(`"exhaustive"`, "exhaustive always present").
			Contains(`"all values are distinct"`, "distinct always present").
			Contains(`"wire compatibility"`, "wire-compat always present").
			NotContains(`"stringer"`, "no stringer").
			NotContains(`"parse round-trip"`, "no parse").
			NotContains(`"marshal text"`, "no text marshal").
			NotContains(`"json round-trip"`, "no json marshal").
			NotContains(`"binary round-trip"`, "no binary marshal")
	})

	t.Run("Source attribution includes invocation flags", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&enum.Generator{}).Generate(pkg, []string{"Status"},
			generator.DefaultConfig(), generator.Options{
				Output:     "status.gen_test.go",
				Invocation: "enum -o status.gen_test.go Status",
			})
		testkit.NoError(t, err, "Generate")
		testkit.Assert(t, string(res.Files[0].Content)).
			Contains("// Source: testkit enum -o status.gen_test.go Status",
				"Source line preserves verbatim flags")
	})

	t.Run("missing type surfaces a hard error (not a silent skip)", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		_, err := (&enum.Generator{}).Generate(pkg, []string{"DoesNotExist"},
			generator.DefaultConfig(), generator.Options{Output: "status.gen_test.go"})
		testkit.True(t, err != nil, "missing type errors")
	})

	t.Run("combined wire golden carries every type's mapping with trailing newline", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&enum.Generator{}).Generate(pkg, []string{"Status", "Priority"},
			generator.DefaultConfig(), generator.Options{Output: "status.gen_test.go"})
		testkit.NoError(t, err, "Generate")

		g := pickFile(res.Files, "status.gen_wire.json")
		body := string(g.Content)
		testkit.True(t, strings.HasSuffix(body, "\n"), "trailing newline")
		testkit.Assert(t, body).
			Contains(`"Status":`, "Status section").
			Contains(`"Priority":`, "Priority section").
			Contains(`"StatusPending": 0`, "Pending = 0").
			Contains(`"StatusActive": 1`, "Active = 1").
			Contains(`"StatusClosed": 2`, "Closed = 2").
			Contains(`"PriorityLow": 0`, "Low = 0").
			Contains(`"PriorityMedium": 1`, "Medium = 1").
			Contains(`"PriorityHigh": 2`, "High = 2")
	})

	t.Run("output is gofmt-clean (header preserved)", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&enum.Generator{}).Generate(pkg, []string{"Status"},
			generator.DefaultConfig(), generator.Options{Output: "status.gen_test.go"})
		testkit.NoError(t, err, "Generate")
		testkit.Assert(t, string(res.Files[0].Content)).
			Contains("// Code generated by testkit enum. DO NOT EDIT.",
				"header preserved")
	})
}

func hasPath(paths []string, want string) bool {
	return slices.Contains(paths, want)
}

// pickFile returns the OutputFile whose Path matches want, or a
// zero-value file if none matches. Used by tests that need to make
// assertions on a single named artifact without scanning the slice
// inline.
func pickFile(files []generator.OutputFile, want string) generator.OutputFile {
	for _, f := range files {
		if f.Path == want {
			return f
		}
	}
	return generator.OutputFile{}
}
