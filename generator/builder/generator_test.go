// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"slices"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/builder"
)

func TestGenerator(t *testing.T) {
	t.Parallel()

	t.Run("Name returns subcommand", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, (&builder.Generator{}).Name(), "builder", "Name")
	})

	t.Run("Generate emits two files: impl + companion test", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "structs")
		res, err := (&builder.Generator{}).Generate(pkg, []string{"Item"},
			generator.DefaultConfig(), generator.Options{
				Output: "structstest/builders.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		testkit.Len(t, res.Files, 2, "impl + test")

		paths := make([]string, len(res.Files))
		for i, f := range res.Files {
			paths[i] = f.Path
		}
		testkit.Equal(t, paths[0], "structstest/builders.gen.go", "impl path")
		testkit.True(t, slices.Contains(paths, "structstest/builders.gen_test.go"),
			"test path is auto-derived via TestPathFrom")
	})

	t.Run("impl emits per-field setters with shape-specific helpers", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "structs")
		res, err := (&builder.Generator{}).Generate(pkg, []string{"Item"},
			generator.DefaultConfig(), generator.Options{
				Output: "structstest/builders.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		impl := string(res.Files[0].Content)
		testkit.Assert(t, impl).
			Contains("type ItemBuilder struct", "builder type emitted").
			Contains("func NewItem()", "constructor").
			Contains("func NewItemFrom(", "from-value constructor").
			Contains("WithID(v string)", "scalar setter").
			Contains("WithTags(v ...string)", "slice variadic setter").
			Contains("AppendTags(", "slice append setter").
			Contains("WithMetadata(m map[string]string)", "map setter").
			Contains("WithMetadataEntry(k string, v string)", "map entry setter").
			Contains("WithData(v []byte)", "bytes setter").
			Contains("WithDataString(s string)", "bytes-string convenience").
			Contains("Mutate(fn", "Mutate hook").
			Contains("Clone()", "Clone").
			Contains("Build()", "Build").
			NotContains("Withhidden", "unexported field MUST NOT get a setter")
	})

	t.Run("test file uses sibling test-package qualifier", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "structs")
		res, err := (&builder.Generator{}).Generate(pkg, []string{"Item"},
			generator.DefaultConfig(), generator.Options{
				Output: "structstest/builders.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		var test string
		for _, f := range res.Files {
			if strings.HasSuffix(f.Path, "_test.go") {
				test = string(f.Content)
			}
		}
		testkit.Assert(t, test).
			Contains("package structstest_test", "external _test package").
			Contains("structstest.NewItem", "calls go through GenQualifier").
			Contains("func TestItemBuilder", "test fn naming")
	})

	t.Run("Holder skips Mutate/Clone subtests when no basic-comparable field", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "structs")
		res, err := (&builder.Generator{}).Generate(pkg, []string{"Holder"},
			generator.DefaultConfig(), generator.Options{
				Output: "structstest/holder.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		var test string
		for _, f := range res.Files {
			if strings.HasSuffix(f.Path, "_test.go") {
				test = string(f.Content)
			}
		}
		testkit.Assert(t, test).
			Contains("func TestHolderBuilder", "test fn emitted").
			NotContains(`"Clone forks independent scalar"`, "no Clone subtest").
			NotContains(`"Mutate modifies value"`, "no Mutate subtest")
	})

	t.Run("generic Container[T] emits parameterized builder + concrete tests", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "generics")
		res, err := (&builder.Generator{}).Generate(pkg, []string{"Container"},
			generator.DefaultConfig(), generator.Options{
				Output: "genericstest/containers.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		impl := string(res.Files[0].Content)
		var test string
		for _, f := range res.Files {
			if strings.HasSuffix(f.Path, "_test.go") {
				test = string(f.Content)
			}
		}
		testkit.Assert(t, impl).
			Contains("type ContainerBuilder[T any] struct", "generic builder type").
			Contains("func NewContainer[T any]()", "generic constructor")
		testkit.Assert(t, test).
			Contains("Container[string]", "concrete instantiation in tests").
			Contains(`NewContainer[string]()`, "test calls concrete constructor")
	})

	t.Run("Numeric-constrained Stat[T] picks int as concrete instantiation", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "generics")
		res, err := (&builder.Generator{}).Generate(pkg, []string{"Stat"},
			generator.DefaultConfig(), generator.Options{
				Output: "genericstest/stats.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		var test string
		for _, f := range res.Files {
			if strings.HasSuffix(f.Path, "_test.go") {
				test = string(f.Content)
			}
		}
		testkit.Assert(t, test).
			Contains("Stat[int]", "constraint-aware: int satisfies Numeric").
			NotContains("Stat[string]", "string fails the Numeric constraint")
	})

	t.Run("Source attribution preserves CLI invocation", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "structs")
		res, err := (&builder.Generator{}).Generate(pkg, []string{"Item"},
			generator.DefaultConfig(), generator.Options{
				Output:     "structstest/builders.gen.go",
				Invocation: "builder -o structstest/builders.gen.go Item",
			})
		testkit.NoError(t, err, "Generate")
		testkit.Assert(t, string(res.Files[0].Content)).
			Contains("// Source: testkit builder -o structstest/builders.gen.go Item",
				"Source line carries verbatim flags")
	})

	t.Run("output is gofmt-clean (header preserved)", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "structs")
		res, err := (&builder.Generator{}).Generate(pkg, []string{"Item"},
			generator.DefaultConfig(), generator.Options{
				Output: "structstest/builders.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		testkit.Assert(t, string(res.Files[0].Content)).
			Contains("// Code generated by testkit builder. DO NOT EDIT.",
				"header preserved")
	})

	t.Run("non-struct args surface a hard error via Pipeline KindStruct gate", func(t *testing.T) {
		t.Parallel()
		// Status is an enum (named int), not a struct — the
		// Pipeline's KindStruct validation must reject it before
		// Analyze runs.
		pkg := loadFixture(t, "basic")
		_, err := (&builder.Generator{}).Generate(pkg, []string{"Status"},
			generator.DefaultConfig(), generator.Options{
				Output: "basictest/builders.gen.go",
			})
		testkit.True(t, err != nil, "non-struct rejected up front")
	})
}
