// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
)

func TestDerivePackageName(t *testing.T) {
	t.Parallel()
	cfg := gen.DefaultConfig()

	t.Run("same directory uses source package", func(t *testing.T) {
		t.Parallel()
		got := gen.DerivePackageName("store.gen.go", "store", cfg)
		testkit.Equal(t, got, "store", "must use source package name")
	})

	t.Run("test file in same dir uses external style", func(t *testing.T) {
		t.Parallel()
		got := gen.DerivePackageName("store.gen_test.go", "store", cfg)
		testkit.Equal(t, got, "store_test", "must append _test")
	})

	t.Run("test file with internal style uses source name", func(t *testing.T) {
		t.Parallel()
		internal := cfg
		internal.TestPackageStyle = gen.TestPackageStyleInternal
		got := gen.DerivePackageName("store.gen_test.go", "store", internal)
		testkit.Equal(t, got, "store", "internal style uses source name")
	})

	t.Run("subdirectory uses directory name", func(t *testing.T) {
		t.Parallel()
		got := gen.DerivePackageName("storetest/in_memory_store.gen.go", "store", cfg)
		testkit.Equal(t, got, "storetest", "must use directory name")
	})
}

func TestTestPathFrom(t *testing.T) {
	t.Parallel()

	t.Run("appends _test.go suffix", func(t *testing.T) {
		t.Parallel()
		got := gen.TestPathFrom("storetest/store.gen.go")
		testkit.Equal(t, got, "storetest/store.gen_test.go", "must append _test.go")
	})

	t.Run("handles root path", func(t *testing.T) {
		t.Parallel()
		got := gen.TestPathFrom("errors.gen.go")
		testkit.Equal(t, got, "errors.gen_test.go", "must work for root path")
	})
}

func TestOutputImportPath(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "basic")

	t.Run("same directory returns package path", func(t *testing.T) {
		t.Parallel()
		got, err := gen.OutputImportPath("store.gen.go", pkg)
		testkit.NoError(t, err, "must succeed")
		testkit.Equal(t, got, pkg.Pkg.Path(), "must return package import path")
	})

	t.Run("subdirectory appends to package path", func(t *testing.T) {
		t.Parallel()
		got, err := gen.OutputImportPath("storetest/foo.gen.go", pkg)
		testkit.NoError(t, err, "must succeed")
		testkit.Assert(t, got).Contains("storetest", "must include subdirectory")
	})
}

func TestValidateTypes(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "basic")

	t.Run("existing interface passes", func(t *testing.T) {
		t.Parallel()
		errs := gen.ValidateTypes(pkg, []string{"Store"}, gen.KindInterface)
		testkit.Len(t, errs, 0, "must pass for valid interface")
	})

	t.Run("missing type fails", func(t *testing.T) {
		t.Parallel()
		errs := gen.ValidateTypes(pkg, []string{"Nonexistent"}, gen.KindInterface)
		testkit.Len(t, errs, 1, "must fail for missing type")
		testkit.Assert(t, errs[0].Message).Contains("not found", "must say not found")
	})

	t.Run("wrong kind fails", func(t *testing.T) {
		t.Parallel()
		errs := gen.ValidateTypes(pkg, []string{"Item"}, gen.KindInterface)
		testkit.Len(t, errs, 1, "must fail for struct when interface expected")
		testkit.Assert(t, errs[0].Message).Contains("not an interface", "must describe mismatch")
	})

	t.Run("struct kind validates structs", func(t *testing.T) {
		t.Parallel()
		errs := gen.ValidateTypes(pkg, []string{"Item"}, gen.KindStruct)
		testkit.Len(t, errs, 0, "must pass for valid struct")
	})

	t.Run("gen.KindAny accepts any named type", func(t *testing.T) {
		t.Parallel()
		errs := gen.ValidateTypes(pkg, []string{"Store", "Item", "Status"}, gen.KindAny)
		testkit.Len(t, errs, 0, "must pass for any named type")
	})

	t.Run("interface when struct expected", func(t *testing.T) {
		t.Parallel()
		errs := gen.ValidateTypes(pkg, []string{"Store"}, gen.KindStruct)
		testkit.Len(t, errs, 1, "must fail for interface when struct expected")
		testkit.Assert(t, errs[0].Message).Contains("not a struct", "must describe mismatch")
	})

	t.Run("multiple errors collected", func(t *testing.T) {
		t.Parallel()
		errs := gen.ValidateTypes(pkg, []string{"Missing1", "Missing2"}, gen.KindInterface)
		testkit.Len(t, errs, 2, "must collect all errors")
	})
}
