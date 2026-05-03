// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enum_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
	"go.thesmos.sh/testkit/gen/enum"
)

func TestGenerate(t *testing.T) {
	t.Parallel()

	t.Run("Name returns enum", func(t *testing.T) {
		t.Parallel()
		g := &enum.Generator{}
		testkit.Equal(t, g.Name(), "enum", "generator name")
	})

	t.Run("produces one output file", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &enum.Generator{}
		result, err := g.Generate(
			pkg, []string{"Status", "Priority"},
			gen.DefaultConfig(), gen.Options{Output: "enum.gen_test.go"},
		)
		testkit.NoError(t, err, "must generate")
		testkit.Len(t, result.Files, 1, "enum produces one file")
	})

	t.Run("output has stringer tests for Status", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &enum.Generator{}
		result, err := g.Generate(
			pkg, []string{"Status"},
			gen.DefaultConfig(), gen.Options{Output: "enum.gen_test.go"},
		)
		testkit.NoError(t, err, "must generate")
		got := string(result.Files[0].Content)
		testkit.Assert(t, got).
			Contains("TestStatusEnum", "test function").
			Contains("stringer", "stringer subtest").
			Contains("out of range", "boundary subtest").
			Contains("exhaustive", "exhaustive subtest").
			Contains("distinct", "distinct subtest")
	})

	t.Run("output skips stringer for Priority", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &enum.Generator{}
		result, err := g.Generate(
			pkg, []string{"Priority"},
			gen.DefaultConfig(), gen.Options{Output: "enum.gen_test.go"},
		)
		testkit.NoError(t, err, "must generate")
		got := string(result.Files[0].Content)
		testkit.Assert(t, got).
			Contains("TestPriorityEnum", "test function").
			Contains("exhaustive", "exhaustive subtest").
			NotContains("stringer", "no stringer for Priority").
			NotContains("out of range", "no boundary for Priority")
	})

	t.Run("nonexistent type produces no output", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &enum.Generator{}
		result, err := g.Generate(
			pkg, []string{"Nonexistent"},
			gen.DefaultConfig(), gen.Options{Output: "enum.gen_test.go"},
		)
		testkit.NoError(t, err, "must succeed")
		testkit.Len(t, result.Files, 0, "no consts means no output")
	})

	t.Run("golden file matches", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &enum.Generator{}
		result, err := g.Generate(
			pkg, []string{"Status", "Priority"},
			gen.DefaultConfig(), gen.Options{Output: "enum.gen_test.go"},
		)
		testkit.NoError(t, err, "must generate")

		goldenPath := filepath.Join(
			testdataDir(t), "basic", "enum.gen_test.go",
		)
		want, readErr := os.ReadFile(goldenPath)
		testkit.NoError(t, readErr, "must read golden file")
		testkit.Equal(t,
			string(result.Files[0].Content), string(want),
			"must match golden",
		)
	})
}
