// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import (
	"testing"

	"go.thesmos.sh/testkit"
)

func TestDerivePackageName(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()

	t.Run("same directory uses source package", func(t *testing.T) {
		t.Parallel()
		got := DerivePackageName("store.gen.go", "store", cfg)
		testkit.Equal(t, got, "store", "must use source package name")
	})

	t.Run("test file in same dir uses external style", func(t *testing.T) {
		t.Parallel()
		got := DerivePackageName("store.gen_test.go", "store", cfg)
		testkit.Equal(t, got, "store_test", "must append _test")
	})

	t.Run("test file with internal style uses source name", func(t *testing.T) {
		t.Parallel()
		internal := cfg
		internal.TestPackageStyle = TestPackageStyleInternal
		got := DerivePackageName("store.gen_test.go", "store", internal)
		testkit.Equal(t, got, "store", "internal style uses source name")
	})

	t.Run("subdirectory uses directory name", func(t *testing.T) {
		t.Parallel()
		got := DerivePackageName("storetest/in_memory_store.gen.go", "store", cfg)
		testkit.Equal(t, got, "storetest", "must use directory name")
	})
}

func TestOutputImportPath(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "basic")

	t.Run("same directory returns package path", func(t *testing.T) {
		t.Parallel()
		got, err := OutputImportPath("store.gen.go", pkg)
		testkit.NoError(t, err, "must succeed")
		testkit.Equal(t, got, pkg.Pkg.Path(), "must return package import path")
	})

	t.Run("subdirectory appends to package path", func(t *testing.T) {
		t.Parallel()
		got, err := OutputImportPath("storetest/foo.gen.go", pkg)
		testkit.NoError(t, err, "must succeed")
		testkit.Assert(t, got).Contains("storetest", "must include subdirectory")
	})
}

func TestValidateTypes(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "basic")

	t.Run("existing interface passes", func(t *testing.T) {
		t.Parallel()
		errs := ValidateTypes(pkg, []string{"Store"}, KindInterface)
		testkit.Len(t, errs, 0, "must pass for valid interface")
	})

	t.Run("missing type fails", func(t *testing.T) {
		t.Parallel()
		errs := ValidateTypes(pkg, []string{"Nonexistent"}, KindInterface)
		testkit.Len(t, errs, 1, "must fail for missing type")
		testkit.Assert(t, errs[0].Message).Contains("not found", "must say not found")
	})

	t.Run("wrong kind fails", func(t *testing.T) {
		t.Parallel()
		errs := ValidateTypes(pkg, []string{"Item"}, KindInterface)
		testkit.Len(t, errs, 1, "must fail for struct when interface expected")
		testkit.Assert(t, errs[0].Message).Contains("not an interface", "must describe mismatch")
	})

	t.Run("struct kind validates structs", func(t *testing.T) {
		t.Parallel()
		errs := ValidateTypes(pkg, []string{"Item"}, KindStruct)
		testkit.Len(t, errs, 0, "must pass for valid struct")
	})

	t.Run("KindAny accepts any named type", func(t *testing.T) {
		t.Parallel()
		errs := ValidateTypes(pkg, []string{"Store", "Item", "Status"}, KindAny)
		testkit.Len(t, errs, 0, "must pass for any named type")
	})

	t.Run("multiple errors collected", func(t *testing.T) {
		t.Parallel()
		errs := ValidateTypes(pkg, []string{"Missing1", "Missing2"}, KindInterface)
		testkit.Len(t, errs, 2, "must collect all errors")
	})
}
