// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
)

// loadBasic loads the testdata/basic fixture used across loader tests.
func loadBasic(t *testing.T) *generator.Package {
	t.Helper()
	loader := generator.NewLoader()
	pkg, err := loader.Load("./testdata/basic", "")
	testkit.NoError(t, err, "Load testdata/basic")
	return pkg
}

func TestLoader(t *testing.T) {
	t.Parallel()

	t.Run("Load caches packages by import path", func(t *testing.T) {
		t.Parallel()
		loader := generator.NewLoader()
		a, err := loader.Load("./testdata/basic", "")
		testkit.NoError(t, err, "first load")
		b, err := loader.Load("./testdata/basic", "")
		testkit.NoError(t, err, "second load")
		testkit.True(t, a == b, "second load returns cached *Package")
	})

	t.Run("Load surfaces failure for unknown packages", func(t *testing.T) {
		t.Parallel()
		_, err := generator.NewLoader().Load("./testdata/does-not-exist", "")
		testkit.True(t, err != nil, "missing package errors")
	})

	t.Run("Path and Name expose package metadata", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		testkit.True(t, pkg.Path() != "", "Path is non-empty")
		testkit.Equal(t, pkg.Name(), "basic", "Name reflects testdata package name")
	})

	t.Run("Interface returns sorted methods + type-level directives", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		iface, err := pkg.Interface("Store")
		testkit.NoError(t, err, "Interface(Store)")

		testkit.Equal(t, iface.Name, "Store", "Name")
		testkit.Len(t, iface.Methods, 2, "two methods")
		testkit.Equal(t, iface.Methods[0].Name, "Get", "alphabetical method order")
		testkit.Equal(t, iface.Methods[1].Name, "Put", "alphabetical method order")

		testkit.Len(t, iface.Directives, 1, "one type-level directive")
		testkit.Equal(t, iface.Directives[0].Name, "idempotent", "//testkit:idempotent")
	})

	t.Run("interface method directives extract via standalone form", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		iface, _ := pkg.Interface("Store")
		get := iface.Methods[0]
		testkit.Equal(t, get.Name, "Get", "Get is first")
		testkit.Len(t, get.Directives, 1, "Get has //testkit:errors")
		testkit.Equal(t, get.Directives[0].Name, "errors", "errors directive")
		testkit.Equal(t, get.Directives[0].Args[0], "ErrNotFound", "ErrNotFound arg")
	})

	t.Run("interface method directives extract via bundle form", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		iface, _ := pkg.Interface("Store")
		put := iface.Methods[1]
		testkit.Equal(t, put.Name, "Put", "Put is second")
		testkit.Len(t, put.Directives, 2, "bundle expanded to 2 directives")
		testkit.Equal(t, put.Directives[0].Name, "atomic", "atomic from bundle")
		testkit.Equal(t, put.Directives[1].Name, "idempotent", "idempotent from bundle")
	})

	t.Run("Struct returns named fields in declaration order", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		s, err := pkg.Struct("Counter")
		testkit.NoError(t, err, "Struct(Counter)")
		testkit.Equal(t, s.Name, "Counter", "Name")
		testkit.Len(t, s.Fields, 2, "two fields")
		testkit.Equal(t, s.Fields[0].Name, "N", "declaration order")
		testkit.Equal(t, s.Fields[1].Name, "Name", "declaration order")
		testkit.Equal(t, s.Fields[1].Tag, `testkit:"required"`, "raw struct tag preserved")
	})

	t.Run("MethodsOn returns concrete-type methods with directives", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		methods := pkg.MethodsOn("Counter")
		testkit.Len(t, methods, 1, "Counter has Reset")
		testkit.Equal(t, methods[0].Name, "Reset", "Reset method")
		testkit.Len(t, methods[0].Directives, 1, "Reset has //testkit:idempotent")
		testkit.Equal(t, methods[0].Directives[0].Name, "idempotent", "idempotent directive")
	})

	t.Run("MethodsOn returns nil for missing or non-named type", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		testkit.True(t, pkg.MethodsOn("DoesNotExist") == nil, "missing → nil")
	})

	t.Run("MethodsOn handles generic-instance pointer receivers", func(t *testing.T) {
		t.Parallel()
		// `*InMemoryHolder[V]` is a *ast.StarExpr whose X is an
		// *ast.IndexExpr — covers recvName's generic-receiver branch.
		pkg := loadFixture(t, "generics")
		methods := pkg.MethodsOn("InMemoryHolder")
		testkit.True(t, len(methods) > 0, "generic struct methods discovered")
	})

	t.Run("Interfaces and Structs list every named type", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		ifaces := pkg.Interfaces()
		testkit.True(t, len(ifaces) >= 1, "at least Store")
		hasStore := false
		for _, i := range ifaces {
			if i.Name == "Store" {
				hasStore = true
			}
		}
		testkit.True(t, hasStore, "Store discoverable via Interfaces()")

		structs := pkg.Structs()
		testkit.True(t, len(structs) >= 1, "at least Counter")
	})

	t.Run("Var and Const resolve package-level declarations", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)

		v, err := pkg.Var("ErrNotFound")
		testkit.NoError(t, err, "Var(ErrNotFound)")
		testkit.Equal(t, v.Name, "ErrNotFound", "Name")

		c, err := pkg.Const("StatusPending")
		testkit.NoError(t, err, "Const(StatusPending)")
		testkit.Equal(t, c.Name, "StatusPending", "Name")
	})

	t.Run("ResolveVar handles unqualified and pkg-qualified names", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)

		v, importPath, err := pkg.ResolveVar("ErrNotFound")
		testkit.NoError(t, err, "ResolveVar(unqualified)")
		testkit.Equal(t, importPath, "", "unqualified has empty importPath")
		testkit.Equal(t, v.Name, "ErrNotFound", "Name")

		// Qualified — pkgName.varName form.
		v, importPath, err = pkg.ResolveVar("io.EOF")
		testkit.NoError(t, err, "ResolveVar(qualified)")
		testkit.Equal(t, importPath, "io", "io import path")
		testkit.Equal(t, v.Name, "EOF", "var name")
	})

	t.Run("ResolveVar errors on missing or unimported packages", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)

		_, _, err := pkg.ResolveVar("io.DoesNotExist")
		testkit.True(t, err != nil, "missing var in imported pkg errors")

		_, _, err = pkg.ResolveVar("not-imported.X")
		testkit.True(t, err != nil, "unimported pkg errors")
	})

	t.Run("Interface lookup fails for missing or wrong-kind types", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		_, err := pkg.Interface("DoesNotExist")
		testkit.True(t, err != nil, "missing interface errors")
		_, err = pkg.Struct("Store")
		testkit.True(t, err != nil, "Store is interface, not struct")
	})

	t.Run("Pkg exposes the underlying go/types package", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		p := pkg.Pkg
		testkit.True(t, p != nil, "Pkg non-nil")
		testkit.Equal(t, p.Name(), "basic", "Pkg.Name matches")
	})

	t.Run("PackageDirectives extracts package-level //testkit: lines", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		dirs := pkg.PackageDirectives()
		// basic/doc.go declares
		// //testkit:sentinel-no-overlap-with go.thesmos.sh/testkit/generator/testdata/storage
		testkit.True(t, len(dirs) > 0, "at least one package directive")
		found := false
		for _, d := range dirs {
			if d.Name == "sentinel-no-overlap-with" {
				found = true
				testkit.True(t, len(d.Args) > 0, "directive has args")
			}
		}
		testkit.True(t, found, "sentinel-no-overlap-with directive present")
	})

	t.Run("PackageDirectives returns empty for packages without directives", func(t *testing.T) {
		t.Parallel()
		pkg, err := generator.NewLoader().Load("./testdata/storage", "")
		testkit.NoError(t, err, "Load testdata/storage")
		dirs := pkg.PackageDirectives()
		testkit.Len(t, dirs, 0, "storage has no package directives")
	})

	t.Run("FieldDirectives returns inline //testkit:default annotations", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "defaults")
		host := pkg.FieldDirectives("Config", "Host")
		testkit.Len(t, host, 1, "Host has one directive")
		testkit.Equal(t, host[0].Name, "default", "name is `default`")
		testkit.Equal(t, host[0].Args[0], `"localhost"`, "arg is the literal default")

		port := pkg.FieldDirectives("Config", "Port")
		testkit.Len(t, port, 1, "Port has one directive")
		testkit.Equal(t, port[0].Args[0], "8080", "Port default is integer literal")
	})

	t.Run("FieldDirectives returns nil for un-annotated fields", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "defaults")
		got := pkg.FieldDirectives("Config", "Name")
		testkit.Len(t, got, 0, "Name has no directive")
	})

	t.Run("FieldDirectives returns nil for missing types and fields", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "defaults")
		testkit.Len(t, pkg.FieldDirectives("DoesNotExist", "X"), 0, "missing type")
		testkit.Len(t, pkg.FieldDirectives("Config", "DoesNotExist"), 0, "missing field")
	})

	t.Run("FieldDirectives returns nil for non-struct types", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		// Status is an enum (named int) — not a struct.
		testkit.Len(t, pkg.FieldDirectives("Status", "X"), 0, "non-struct → nil")
	})

	t.Run("Path and Name return empty when Pkg is nil", func(t *testing.T) {
		t.Parallel()
		var p generator.Package
		testkit.Equal(t, p.Path(), "", "no Pkg → empty path")
		testkit.Equal(t, p.Name(), "", "no Pkg → empty name")
	})

	t.Run("Var rejects non-variable lookups", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		_, err := pkg.Var("DoesNotExist")
		testkit.True(t, err != nil, "missing var errors")
		// Status is a type, not a var — Var must reject it.
		_, err = pkg.Var("Status")
		testkit.True(t, err != nil, "type-name rejected")
		testkit.Assert(t, err.Error()).Contains("not a variable", "diagnostic")
	})

	t.Run("Const rejects non-constant lookups", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		_, err := pkg.Const("DoesNotExist")
		testkit.True(t, err != nil, "missing const errors")
		// ErrNotFound is a var, not a const.
		_, err = pkg.Const("ErrNotFound")
		testkit.True(t, err != nil, "var-name rejected")
		testkit.Assert(t, err.Error()).Contains("not a constant", "diagnostic")
	})

	t.Run("Struct rejects non-type and non-named lookups", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		// ErrNotFound is a var, not a type.
		_, err := pkg.Struct("ErrNotFound")
		testkit.True(t, err != nil, "var-name rejected")
		testkit.Assert(t, err.Error()).Contains("not a type", "diagnostic")
	})

	t.Run("Interface rejects non-type and non-interface lookups", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		// ErrNotFound is a var, not a type.
		_, err := pkg.Interface("ErrNotFound")
		testkit.True(t, err != nil, "var-name rejected")
		testkit.Assert(t, err.Error()).Contains("not a type", "diagnostic")
	})
}
