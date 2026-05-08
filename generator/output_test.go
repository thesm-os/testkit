// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
)

func TestTestPathFrom(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"gen.go suffix", "storetest/store_stub.gen.go", "storetest/store_stub.gen_test.go"},
		{"already a test file", "storetest/store_stub.gen_test.go", "storetest/store_stub.gen_test.go"},
		{"plain .go suffix", "foo.go", "foo_test.go"},
		{"_test.go is unchanged", "foo_test.go", "foo_test.go"},
		{"path with dots in directory", "path/with.dots/sentinel.gen.go", "path/with.dots/sentinel.gen_test.go"},
		{"no extension", "plainname", "plainname_test.go"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testkit.Equal(t, generator.TestPathFrom(tc.in), tc.want, "TestPathFrom("+tc.in+")")
		})
	}
}

func TestBuildTestFileInfo(t *testing.T) {
	t.Parallel()

	t.Run("external style adds gen import and qualifier", func(t *testing.T) {
		t.Parallel()
		cfg := generator.DefaultConfig() // External
		info := generator.BuildTestFileInfo("store", []generator.Import{{Path: "io"}}, cfg, "example.com/x/storetest")

		testkit.Equal(t, info.PackageName, "store_test", "external package suffix")
		testkit.Equal(t, info.GenQualifier, "storetest.", "qualifier matches base")

		hasGenImport := false
		for _, imp := range info.Imports {
			if imp.Path == "example.com/x/storetest" {
				hasGenImport = true
			}
		}
		testkit.True(t, hasGenImport, "external test must import the gen package")
	})

	t.Run("internal style keeps source package name", func(t *testing.T) {
		t.Parallel()
		cfg := generator.DefaultConfig()
		cfg.TestPackageStyle = generator.TestPackageStyleInternal
		info := generator.BuildTestFileInfo("store", nil, cfg, "example.com/x/storetest")

		testkit.Equal(t, info.PackageName, "store", "internal style stays in source pkg")
		testkit.Equal(t, info.GenQualifier, "", "internal style has no qualifier")
	})
}

func TestOutputImportPath(t *testing.T) {
	t.Parallel()

	t.Run("computes import path from module-aware package", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		// Output in the same dir as the source — should resolve to the
		// fixture's own import path.
		got, err := generator.OutputImportPath("out.gen.go", pkg, generator.Options{WorkDir: ""})
		testkit.NoError(t, err, "OutputImportPath")
		testkit.True(t, got != "", "non-empty import path")
	})

	t.Run("nil package surfaces an error", func(t *testing.T) {
		t.Parallel()
		_, err := generator.OutputImportPath("out.go", nil, generator.Options{})
		testkit.True(t, err != nil, "nil package errors")
	})

	t.Run("OutputImportBase short-circuits module resolution", func(t *testing.T) {
		t.Parallel()
		// When the CLI loads a remote package via -p, it sets
		// OutputImportBase to the CWD's import path. The output
		// resolves relative to that, not the source pkg's module.
		got, err := generator.OutputImportPath("out.go", nil, generator.Options{
			OutputImportBase: "example.com/cwd",
		})
		testkit.NoError(t, err, "OutputImportPath with OutputImportBase")
		testkit.Equal(t, got, "example.com/cwd", "same-dir → bare base")
	})

	t.Run("OutputImportBase composes with subdirectory output paths", func(t *testing.T) {
		t.Parallel()
		got, err := generator.OutputImportPath("storetest/foo.gen.go", nil, generator.Options{
			OutputImportBase: "example.com/cwd",
		})
		testkit.NoError(t, err, "OutputImportPath with subdir")
		testkit.Equal(t, got, "example.com/cwd/storetest", "subdir appends to base")
	})
}

func TestBuildOutputCtx(t *testing.T) {
	t.Parallel()

	t.Run("subdir output sets ImportPath + Qualifier", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		ctx, err := generator.BuildOutputCtx(pkg, generator.DefaultConfig(),
			generator.Options{Output: "storetest/foo.gen.go"})
		testkit.NoError(t, err, "BuildOutputCtx")
		testkit.Equal(t, ctx.PackageName, "storetest", "subdir basename")
		testkit.True(t, ctx.ImportPath != "", "imports source pkg")
		testkit.Equal(t, ctx.Qualifier, "basic.", "dotted qualifier")
		testkit.True(t, ctx.Tracker != nil, "tracker primed")
	})

	t.Run("same-package output skips ImportPath + Qualifier", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		// WorkDir scopes OutputImportPath to the basic package's own
		// dir, so foo.gen.go lands in the same package and no import
		// is needed.
		ctx, err := generator.BuildOutputCtx(pkg, generator.DefaultConfig(),
			generator.Options{Output: "foo.gen.go", WorkDir: "testdata/basic"})
		testkit.NoError(t, err, "BuildOutputCtx")
		testkit.Equal(t, ctx.PackageName, "basic", "source pkg name")
		testkit.Equal(t, ctx.ImportPath, "", "no import needed")
		testkit.Equal(t, ctx.Qualifier, "", "no qualifier needed")
	})

	t.Run("external test file in source dir gets _test suffix and import", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		ctx, err := generator.BuildOutputCtx(pkg, generator.DefaultConfig(),
			generator.Options{Output: "errors.gen_test.go"})
		testkit.NoError(t, err, "BuildOutputCtx")
		testkit.Equal(t, ctx.PackageName, "basic_test", "external _test pkg")
		testkit.True(t, ctx.ImportPath != "", "imports source pkg")
		testkit.Equal(t, ctx.Qualifier, "basic.", "dotted qualifier")
	})

	t.Run("OutputPackage CLI override replaces source pkg name", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		ctx, err := generator.BuildOutputCtx(pkg, generator.DefaultConfig(),
			generator.Options{Output: "foo.gen.go", OutputPackage: "remote"})
		testkit.NoError(t, err, "BuildOutputCtx")
		testkit.Equal(t, ctx.PackageName, "remote", "OutputPackage overrides")
	})
}

func TestDerivePackageName(t *testing.T) {
	t.Parallel()

	t.Run("test file in source dir uses external suffix by default", func(t *testing.T) {
		t.Parallel()
		got := generator.DerivePackageName("errors.gen_test.go", "basic", generator.DefaultConfig())
		testkit.Equal(t, got, "basic_test", "external style appends suffix")
	})

	t.Run("test file in source dir uses source pkg under internal style", func(t *testing.T) {
		t.Parallel()
		cfg := generator.DefaultConfig()
		cfg.TestPackageStyle = generator.TestPackageStyleInternal
		got := generator.DerivePackageName("errors.gen_test.go", "basic", cfg)
		testkit.Equal(t, got, "basic", "internal style stays in source pkg")
	})

	t.Run("non-test file in source dir uses source pkg", func(t *testing.T) {
		t.Parallel()
		got := generator.DerivePackageName("foo.gen.go", "basic", generator.DefaultConfig())
		testkit.Equal(t, got, "basic", "source pkg name")
	})

	t.Run("file in subdirectory uses subdir basename", func(t *testing.T) {
		t.Parallel()
		got := generator.DerivePackageName("storetest/store_stub.gen.go", "basic", generator.DefaultConfig())
		testkit.Equal(t, got, "storetest", "subdir basename")
	})
}

func TestValidateTypes(t *testing.T) {
	t.Parallel()

	pkg := loadBasic(t)

	t.Run("KindAny accepts any named type", func(t *testing.T) {
		t.Parallel()
		errs := generator.ValidateTypes(pkg, []string{"Store", "Counter"}, generator.KindAny)
		testkit.Len(t, errs, 0, "any kind accepts both")
	})

	t.Run("KindInterface rejects struct args", func(t *testing.T) {
		t.Parallel()
		errs := generator.ValidateTypes(pkg, []string{"Counter"}, generator.KindInterface)
		testkit.True(t, len(errs) > 0, "Counter is struct, not interface")
	})

	t.Run("KindStruct rejects interface args", func(t *testing.T) {
		t.Parallel()
		errs := generator.ValidateTypes(pkg, []string{"Store"}, generator.KindStruct)
		testkit.True(t, len(errs) > 0, "Store is interface, not struct")
	})

	t.Run("missing type errors", func(t *testing.T) {
		t.Parallel()
		errs := generator.ValidateTypes(pkg, []string{"DoesNotExist"}, generator.KindAny)
		testkit.True(t, len(errs) > 0, "missing type errors")
	})

	t.Run("KindNamedType accepts any named decl", func(t *testing.T) {
		t.Parallel()
		errs := generator.ValidateTypes(pkg, []string{"Status"}, generator.KindNamedType)
		testkit.Len(t, errs, 0, "Status is named type")
	})
}
