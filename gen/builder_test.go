// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import (
	"path/filepath"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
)

func TestGenerateBuilder(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "builder")
	cfg := DefaultConfig()

	t.Run("Account struct with all field types", func(t *testing.T) {
		t.Parallel()
		result, err := GenerateBuilder(pkg, []string{"Account"}, cfg, Options{
			Output: "buildertest/account_builder.gen.go",
		})
		testkit.NoError(t, err, "generation must succeed")
		testkit.Len(t, result.Files, 2, "must produce impl + test files")

		impl := string(result.Files[0].Content)

		// Basic type setters.
		testkit.Assert(t, impl).
			Contains("WithName", "must have string setter").
			Contains("WithAge", "must have int setter").
			Contains("WithBalance", "must have float64 setter").
			Contains("WithActive", "must have bool setter")

		// Named type setters.
		testkit.Assert(t, impl).
			Contains("WithAccountID", "must have named string setter").
			Contains("WithStatus", "must have named int setter")

		// Pointer type setters.
		testkit.Assert(t, impl).
			Contains("WithNickname", "must have *string setter").
			Contains("WithManager", "must have *Account setter").
			Contains("WithScore", "must have *int setter")

		// Nested struct setter.
		testkit.Assert(t, impl).Contains("WithAddress", "must have nested struct setter")

		// Stdlib type setters.
		testkit.Assert(t, impl).
			Contains("WithCreatedAt", "must have time.Time setter").
			Contains("WithTTL", "must have time.Duration setter")

		// Slice setters.
		testkit.Assert(t, impl).
			Contains("WithTags", "must have []string setter").
			Contains("WithScores", "must have []int setter").
			Contains("WithChildren", "must have []*Account setter")

		// Map setters.
		testkit.Assert(t, impl).
			Contains("WithMetadata", "must have map[string]string setter").
			Contains("WithCounts", "must have map[string]int setter")

		// Array setter.
		testkit.Assert(t, impl).Contains("WithChecksum", "must have [32]byte setter")

		// Interface setter.
		testkit.Assert(t, impl).Contains("WithLogger", "must have io.Writer setter")

		// Function setter.
		testkit.Assert(t, impl).Contains("WithOnChange", "must have func setter")

		// Channel setter.
		testkit.Assert(t, impl).Contains("WithEvents", "must have chan setter")

		// Empty struct setter.
		testkit.Assert(t, impl).Contains("WithMarker", "must have struct{} setter")

		// Must skip unexported fields.
		testkit.Assert(t, impl).NotContains("Withinternal", "must skip unexported fields")

		// Must have both constructors.
		testkit.Assert(t, impl).
			Contains("NewAccountBuilder", "must have seeded constructor").
			Contains("NewDefaultAccountBuilder", "must have zero-value constructor")
	})

	t.Run("generated tests have real assertions for value fields", func(t *testing.T) {
		t.Parallel()
		result, err := GenerateBuilder(pkg, []string{"Account"}, cfg, Options{
			Output: "buildertest/account_builder.gen.go",
		})
		testkit.NoError(t, err, "generation must succeed")
		test := string(result.Files[1].Content)

		// Value fields must have testkit.Equal assertions with sample values.
		testkit.Assert(t, test).
			Contains(`"test-name"`, "must have sample string for Name").
			Contains("42", "must have sample int for Age").
			Contains("3.14", "must have sample float for Balance").
			Contains("testkit.Equal", "must use testkit.Equal")
	})

	t.Run("generated tests handle pointer fields safely", func(t *testing.T) {
		t.Parallel()
		result, err := GenerateBuilder(pkg, []string{"Account"}, cfg, Options{
			Output: "buildertest/account_builder.gen.go",
		})
		testkit.NoError(t, err, "generation must succeed")
		test := string(result.Files[1].Content)

		// Pointer fields must use &sample and *got.Field pattern.
		testkit.Assert(t, test).
			Contains("&sample", "must pass address for pointer fields").
			Contains("== nil", "must nil-check pointer fields before deref")
	})

	t.Run("generated tests handle untestable fields gracefully", func(t *testing.T) {
		t.Parallel()
		result, err := GenerateBuilder(pkg, []string{"Account"}, cfg, Options{
			Output: "buildertest/account_builder.gen.go",
		})
		testkit.NoError(t, err, "generation must succeed")
		test := string(result.Files[1].Content)

		// Interface, func, and chan fields should have compile-only tests.
		testkit.Assert(t, test).
			Contains("WithLogger compiles", "must have compile-only test for interface").
			Contains("WithOnChange compiles", "must have compile-only test for func").
			Contains("WithEvents compiles", "must have compile-only test for chan")
	})

	t.Run("Minimal struct with one field", func(t *testing.T) {
		t.Parallel()
		result, err := GenerateBuilder(pkg, []string{"Minimal"}, cfg, Options{
			Output: "buildertest/minimal_builder.gen.go",
		})
		testkit.NoError(t, err, "generation must succeed")
		impl := string(result.Files[0].Content)
		testkit.Assert(t, impl).
			Contains("MinimalBuilder", "must have builder").
			Contains("WithValue", "must have the one setter")
	})

	t.Run("Empty struct with no exported fields", func(t *testing.T) {
		t.Parallel()
		result, err := GenerateBuilder(pkg, []string{"Empty"}, cfg, Options{
			Output: "buildertest/empty_builder.gen.go",
		})
		testkit.NoError(t, err, "generation must succeed")
		impl := string(result.Files[0].Content)
		testkit.Assert(t, impl).
			Contains("EmptyBuilder", "must have builder type").
			Contains("Build()", "must have Build method").
			NotContains("func (b *EmptyBuilder) With", "must have no setters")
	})

	t.Run("validates type exists", func(t *testing.T) {
		t.Parallel()
		_, err := GenerateBuilder(pkg, []string{"Nonexistent"}, cfg, Options{
			Output: "buildertest/nonexistent.gen.go",
		})
		testkit.Error(t, err, "must fail for missing type")
	})

	t.Run("validates type is struct not interface", func(t *testing.T) {
		t.Parallel()
		basicPkg := loadTestPackage(t, "basic")
		_, err := GenerateBuilder(basicPkg, []string{"Store"}, cfg, Options{
			Output: "basictest/store.gen.go",
		})
		testkit.Error(t, err, "must fail for interface type")
	})

	t.Run("multiple types in one file", func(t *testing.T) {
		t.Parallel()
		result, err := GenerateBuilder(pkg, []string{"Account", "Address"}, cfg, Options{
			Output: "buildertest/builders.gen.go",
		})
		testkit.NoError(t, err, "generation must succeed")
		impl := string(result.Files[0].Content)
		testkit.Assert(t, impl).
			Contains("AccountBuilder", "must have AccountBuilder").
			Contains("AddressBuilder", "must have AddressBuilder")
	})

	t.Run("impl has correct package name", func(t *testing.T) {
		t.Parallel()
		result, err := GenerateBuilder(pkg, []string{"Minimal"}, cfg, Options{
			Output: "buildertest/minimal_builder.gen.go",
		})
		testkit.NoError(t, err, "generation must succeed")
		testkit.Assert(t, string(result.Files[0].Content)).
			Contains("package buildertest", "must have correct package")
	})

	t.Run("test file has _test package suffix", func(t *testing.T) {
		t.Parallel()
		result, err := GenerateBuilder(pkg, []string{"Minimal"}, cfg, Options{
			Output: "buildertest/minimal_builder.gen.go",
		})
		testkit.NoError(t, err, "generation must succeed")
		testkit.Assert(t, string(result.Files[1].Content)).
			Contains("package buildertest_test", "must have test package")
	})

	t.Run("both files have DO NOT EDIT header", func(t *testing.T) {
		t.Parallel()
		result, err := GenerateBuilder(pkg, []string{"Minimal"}, cfg, Options{
			Output: "buildertest/minimal_builder.gen.go",
		})
		testkit.NoError(t, err, "generation must succeed")
		for _, f := range result.Files {
			testkit.Assert(t, string(f.Content)).
				Contains("DO NOT EDIT", "must have DO NOT EDIT marker")
		}
	})

	t.Run("test file path derives from impl path", func(t *testing.T) {
		t.Parallel()
		result, err := GenerateBuilder(pkg, []string{"Minimal"}, cfg, Options{
			Output: "buildertest/minimal_builder.gen.go",
		})
		testkit.NoError(t, err, "generation must succeed")
		testkit.Assert(t, result.Files[1].Path).Contains("_test.go", "must end with _test.go")
	})

	t.Run("uses default output path when -o not specified", func(t *testing.T) {
		t.Parallel()
		result, err := GenerateBuilder(pkg, []string{"Account"}, cfg, Options{})
		testkit.NoError(t, err, "generation must succeed")
		testkit.True(t, strings.Contains(result.Files[0].Path, "account_builder"),
			"must derive path from type name")
	})

	t.Run("same-package output has no qualifier on types", func(t *testing.T) {
		t.Parallel()
		result, err := GenerateBuilder(pkg, []string{"Account"}, cfg, Options{
			Output: "types_builder.gen.go",
		})
		testkit.NoError(t, err, "generation must succeed")
		impl := string(result.Files[0].Content)
		testkit.Assert(t, impl).
			NotContains("builder.Account", "same-package types must not be qualified")
	})

	//nolint:paralleltest // writes fixture files to testdata
	t.Run("generate same-package fixtures", func(t *testing.T) {
		result, err := GenerateBuilder(pkg, []string{"Account", "Address", "Minimal"}, cfg, Options{
			Output: "types_builder.gen.go",
		})
		testkit.NoError(t, err, "generation must succeed")
		err = WriteResult(result, filepath.Join(testdataDir(t), "builder"), false)
		testkit.NoError(t, err, "writing generated files must succeed")
	})

	//nolint:paralleltest // writes fixture files to testdata
	t.Run("generate cross-package fixtures", func(t *testing.T) {
		result, err := GenerateBuilder(pkg, []string{"Account", "Address"}, cfg, Options{
			Output: "buildertest/account_builder.gen.go",
		})
		testkit.NoError(t, err, "generation must succeed")

		// Verify cross-package qualifiers.
		impl := string(result.Files[0].Content)
		testkit.Assert(t, impl).
			Contains("package buildertest", "must use subdirectory package name").
			Contains("builder.Account", "must qualify source types with package name").
			Contains("builder.Address", "must qualify nested struct types")

		test := string(result.Files[1].Content)
		testkit.Assert(t, test).
			Contains("package buildertest_test", "must use external test package").
			Contains("builder.Account", "tests must qualify source types")

		// Write and verify it compiles.
		err = WriteResult(result, filepath.Join(testdataDir(t), "builder"), false)
		testkit.NoError(t, err, "writing cross-package files must succeed")
	})

	t.Run("cross-package output imports source and stdlib types", func(t *testing.T) {
		t.Parallel()
		result, err := GenerateBuilder(pkg, []string{"Account"}, cfg, Options{
			Output: "buildertest/account_builder.gen.go",
		})
		testkit.NoError(t, err, "generation must succeed")
		impl := string(result.Files[0].Content)
		// Account has time.Time and io.Writer fields — these need imports.
		testkit.Assert(t, impl).
			Contains(`"time"`, "must import time for time.Time field").
			Contains(`"io"`, "must import io for io.Writer field")
	})

	t.Run("output is deterministic across runs", func(t *testing.T) {
		t.Parallel()
		opts := Options{Output: "buildertest/account_builder.gen.go"}
		r1, err := GenerateBuilder(pkg, []string{"Account"}, cfg, opts)
		testkit.NoError(t, err, "first run")
		r2, err := GenerateBuilder(pkg, []string{"Account"}, cfg, opts)
		testkit.NoError(t, err, "second run")
		for i := range r1.Files {
			testkit.Equal(t, string(r1.Files[i].Content), string(r2.Files[i].Content),
				"output must be identical across runs")
		}
	})
}
