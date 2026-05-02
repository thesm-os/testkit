// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import (
	"path/filepath"
	"testing"

	"go.thesmos.sh/testkit"
)

func TestGenerateEnum(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "enum")
	cfg := DefaultConfig()

	t.Run("generates tests for Status enum", func(t *testing.T) {
		t.Parallel()
		result, err := GenerateEnum(pkg, []string{"Status"}, cfg, Options{})
		testkit.NoError(t, err, "generation must succeed")
		testkit.Len(t, result.Files, 1, "must produce one test file")

		content := string(result.Files[0].Content)
		testkit.Assert(t, content).
			Contains("TestStatus_Enum", "must have test function named after type").
			Contains("StatusUnspecified", "must reference all constants").
			Contains("StatusPending", "must reference all constants").
			Contains("StatusActive", "must reference all constants").
			Contains("StatusClosed", "must reference all constants")
	})

	t.Run("has all five subtests", func(t *testing.T) {
		t.Parallel()
		result, err := GenerateEnum(pkg, []string{"Status"}, cfg, Options{})
		testkit.NoError(t, err, "generation must succeed")
		content := string(result.Files[0].Content)
		testkit.Assert(t, content).
			Contains(`"stringer"`, "must have stringer subtest").
			Contains(`"exhaustive"`, "must have exhaustive subtest").
			Contains(`"boundary max+1 uses fallback format"`, "must have boundary subtest").
			Contains(`"all values are distinct"`, "must have distinct subtest")
	})

	t.Run("multiple types in one file", func(t *testing.T) {
		t.Parallel()
		result, err := GenerateEnum(pkg, []string{"Status", "Priority"}, cfg, Options{
			Output: "enums.gen_test.go",
		})
		testkit.NoError(t, err, "generation must succeed")
		content := string(result.Files[0].Content)
		testkit.Assert(t, content).
			Contains("TestStatus_Enum", "must have Status test").
			Contains("TestPriority_Enum", "must have Priority test")
	})

	t.Run("no inline comments falls back to constant name", func(t *testing.T) {
		t.Parallel()
		result, err := GenerateEnum(pkg, []string{"Region"}, cfg, Options{
			Output: "region_enum.gen_test.go",
		})
		testkit.NoError(t, err, "generation must succeed")
		content := string(result.Files[0].Content)
		testkit.Assert(t, content).
			Contains(`"RegionUS"`, "must fall back to constant name for RegionUS").
			Contains(`"RegionEU"`, "must fall back to constant name for RegionEU").
			Contains(`"RegionAP"`, "must fall back to constant name for RegionAP")
	})

	t.Run("fails on type with no constants", func(t *testing.T) {
		t.Parallel()
		basicPkg := loadTestPackage(t, "basic")
		_, err := GenerateEnum(basicPkg, []string{"Item"}, cfg, Options{})
		testkit.Error(t, err, "must fail when type has no constants")
	})

	t.Run("fails on nonexistent type", func(t *testing.T) {
		t.Parallel()
		_, err := GenerateEnum(pkg, []string{"Nonexistent"}, cfg, Options{})
		testkit.Error(t, err, "must fail for missing type")
	})

	t.Run("has DO NOT EDIT header", func(t *testing.T) {
		t.Parallel()
		result, err := GenerateEnum(pkg, []string{"Status"}, cfg, Options{})
		testkit.NoError(t, err, "generation must succeed")
		testkit.Assert(t, string(result.Files[0].Content)).
			Contains("DO NOT EDIT", "must have generated header")
	})

	t.Run("output is deterministic", func(t *testing.T) {
		t.Parallel()
		r1, err := GenerateEnum(pkg, []string{"Status"}, cfg, Options{})
		testkit.NoError(t, err, "first run")
		r2, err := GenerateEnum(pkg, []string{"Status"}, cfg, Options{})
		testkit.NoError(t, err, "second run")
		testkit.Equal(t, string(r1.Files[0].Content), string(r2.Files[0].Content),
			"output must be identical across runs")
	})

	//nolint:paralleltest // writes fixture files to testdata
	t.Run("generate fixtures as compilation proof", func(t *testing.T) {
		result, err := GenerateEnum(pkg, []string{"Status", "Priority", "Region"}, cfg, Options{
			Output: "status_enum.gen_test.go",
		})
		testkit.NoError(t, err, "generation must succeed")
		err = WriteResult(result, filepath.Join(testdataDir(t), "enum"), false)
		testkit.NoError(t, err, "writing generated files must succeed")
	})
}
