// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sentinel_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
	"go.thesmos.sh/testkit/gen/sentinel"
)

func TestGenerate(t *testing.T) {
	t.Parallel()

	t.Run("Name returns sentinel", func(t *testing.T) {
		t.Parallel()
		g := &sentinel.Generator{}
		testkit.Equal(t, g.Name(), "sentinel", "generator name")
	})

	t.Run("produces one output file", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &sentinel.Generator{}
		result, err := g.Generate(pkg, nil, gen.DefaultConfig(), gen.Options{
			Output: "errors.gen_test.go",
		})
		testkit.NoError(t, err, "must generate")
		testkit.Len(t, result.Files, 1, "sentinel produces one file")
	})

	t.Run("output has expected content", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &sentinel.Generator{}
		result, err := g.Generate(pkg, nil, gen.DefaultConfig(), gen.Options{
			Output: "errors.gen_test.go",
		})
		testkit.NoError(t, err, "must generate")
		got := string(result.Files[0].Content)
		testkit.Assert(t, got).
			Contains("TestBasicSentinelErrors", "test function name").
			Contains("ErrNotFound", "sentinel var").
			Contains("ErrConflict", "sentinel var").
			Contains("ErrForbidden", "sentinel var").
			Contains("prefix", "prefix subtest").
			Contains("uniqueness", "uniqueness subtest").
			Contains("non-overlap", "non-overlap subtest").
			Contains("unwrap chain", "unwrap chain subtest").
			Contains("TestValidationError", "error type test").
			Contains("errors.As", "errors.As test for custom type")
	})

	t.Run("source file filter limits sentinel vars", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &sentinel.Generator{}
		// SourceFile filter limits Err* vars but error types are still found.
		result, err := g.Generate(pkg, nil, gen.DefaultConfig(), gen.Options{
			Output:     "errors.gen_test.go",
			SourceFile: "nonexistent.go",
		})
		testkit.NoError(t, err, "must succeed")
		// Still has output because ValidationError is an error type.
		testkit.Len(t, result.Files, 1, "error types still produce output")
		got := string(result.Files[0].Content)
		testkit.Assert(t, got).
			NotContains("ErrNotFound", "filtered vars must not appear").
			Contains("ValidationError", "error types must still appear")
	})

	t.Run("golden file matches", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &sentinel.Generator{}
		result, err := g.Generate(pkg, nil, gen.DefaultConfig(), gen.Options{
			Output: "errors.gen_test.go",
		})
		testkit.NoError(t, err, "must generate")

		goldenPath := filepath.Join(testdataDir(t), "basic", "errors.gen_test.go")
		want, readErr := os.ReadFile(goldenPath)
		testkit.NoError(t, readErr, "must read golden file")
		testkit.Equal(t, string(result.Files[0].Content), string(want), "must match golden")
	})
}
