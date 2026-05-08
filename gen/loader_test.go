// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen_test

import (
	"path/filepath"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
)

func TestLoader(t *testing.T) {
	t.Parallel()

	t.Run("caches by import path", func(t *testing.T) {
		t.Parallel()
		loader := gen.NewLoader()
		dir := filepath.Join(testdataDir(t), "basic")
		p1, err := loader.Load(".", dir)
		testkit.NoError(t, err, "first load")
		p2, err := loader.Load(".", dir)
		testkit.NoError(t, err, "second load")
		testkit.True(t, p1 == p2, "must return same pointer from cache")
	})

	t.Run("returns error for nonexistent package", func(t *testing.T) {
		t.Parallel()
		loader := gen.NewLoader()
		_, err := loader.Load(".", "/nonexistent/path/to/package")
		testkit.Error(t, err, "must fail for nonexistent package")
	})
}

func TestPackage(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "basic")

	t.Run("Interface loads and sorts methods", func(t *testing.T) {
		t.Parallel()
		iface, err := pkg.Interface("Store")
		testkit.NoError(t, err, "must find Store")
		testkit.Equal(t, iface.Name, "Store", "name must match")
		testkit.Len(t, iface.Methods, 7, "must have 7 methods")

		names := make([]string, len(iface.Methods))
		for i, m := range iface.Methods {
			names[i] = m.Name
		}
		testkit.Equal(t, names,
			[]string{"Count", "Delete", "Find", "Get", "LegacyPut", "Ping", "Put"},
			"must be sorted")
	})

	t.Run("Interface extracts method doc comments", func(t *testing.T) {
		t.Parallel()
		iface, err := pkg.Interface("Store")
		testkit.NoError(t, err, "must find Store")
		var getDoc string
		for _, m := range iface.Methods {
			if m.Name == "Get" {
				getDoc = m.Doc
			}
		}
		testkit.Assert(t, getDoc).Contains("retrieves an item", "must extract doc")
	})

	t.Run("Interface extracts type doc comment", func(t *testing.T) {
		t.Parallel()
		iface, err := pkg.Interface("Store")
		testkit.NoError(t, err, "must find Store")
		testkit.Assert(t, iface.Doc).Contains("manages items", "must extract type doc")
	})

	t.Run("Interface not found returns error", func(t *testing.T) {
		t.Parallel()
		_, err := pkg.Interface("Nonexistent")
		testkit.Error(t, err, "must fail for missing type")
	})

	t.Run("Interface on struct returns error", func(t *testing.T) {
		t.Parallel()
		_, err := pkg.Interface("Item")
		testkit.Error(t, err, "must fail for struct")
	})

	t.Run("Interface has no type params for non-generic", func(t *testing.T) {
		t.Parallel()
		iface, err := pkg.Interface("Store")
		testkit.NoError(t, err, "must find Store")
		testkit.Len(t, iface.TypeParams, 0, "non-generic has no type params")
	})

	t.Run("Struct loads fields with export status", func(t *testing.T) {
		t.Parallel()
		s, err := pkg.Struct("Item")
		testkit.NoError(t, err, "must find Item")
		testkit.Equal(t, s.Name, "Item", "name must match")
		testkit.Len(t, s.Fields, 12, "must have 12 fields including unexported")

		var exported, unexported int
		for _, f := range s.Fields {
			if f.Exported {
				exported++
			} else {
				unexported++
			}
		}
		testkit.Equal(t, exported, 11, "11 exported fields")
		testkit.Equal(t, unexported, 1, "1 unexported field")
	})

	t.Run("Struct not found returns error", func(t *testing.T) {
		t.Parallel()
		_, err := pkg.Struct("Nonexistent")
		testkit.Error(t, err, "must fail for missing type")
	})

	t.Run("Struct on interface returns error", func(t *testing.T) {
		t.Parallel()
		_, err := pkg.Struct("Store")
		testkit.Error(t, err, "must fail for interface")
	})

	t.Run("Var loads with doc comment", func(t *testing.T) {
		t.Parallel()
		v, err := pkg.Var("ErrNotFound")
		testkit.NoError(t, err, "must find ErrNotFound")
		testkit.Equal(t, v.Name, "ErrNotFound", "name must match")
		testkit.Assert(t, v.Doc).Contains("not found", "must extract doc")
	})

	t.Run("Var not found returns error", func(t *testing.T) {
		t.Parallel()
		_, err := pkg.Var("Nonexistent")
		testkit.Error(t, err, "must fail")
	})

	t.Run("Const loads value", func(t *testing.T) {
		t.Parallel()
		c, err := pkg.Const("StatusPending")
		testkit.NoError(t, err, "must find StatusPending")
		testkit.Equal(t, c.Name, "StatusPending", "name must match")
	})

	t.Run("Const not found returns error", func(t *testing.T) {
		t.Parallel()
		_, err := pkg.Const("Nonexistent")
		testkit.Error(t, err, "must fail")
	})

	t.Run("ErrorVars returns sorted exported Err* vars", func(t *testing.T) {
		t.Parallel()
		vars := pkg.ErrorVars()
		names := make([]string, len(vars))
		for i, v := range vars {
			names[i] = v.Name
		}
		testkit.Equal(t, names, []string{"ErrConflict", "ErrNotFound"}, "must be sorted")
	})

	t.Run("ResolveVar with bare name resolves locally", func(t *testing.T) {
		t.Parallel()
		v, importPath, err := pkg.ResolveVar("ErrNotFound")
		testkit.NoError(t, err, "must resolve")
		testkit.Equal(t, v.Name, "ErrNotFound", "name must match")
		testkit.Equal(t, importPath, "", "local var has empty import path")
	})

	t.Run("ResolveVar with qualified nonexistent package returns error", func(t *testing.T) {
		t.Parallel()
		_, _, err := pkg.ResolveVar("nopkg.ErrFoo")
		testkit.Error(t, err, "must fail for unknown package")
	})

	t.Run("ResolveVar with bare nonexistent name returns error", func(t *testing.T) {
		t.Parallel()
		_, _, err := pkg.ResolveVar("ErrNonexistent")
		testkit.Error(t, err, "must fail for unknown var")
	})

	t.Run("ErrorVars with file filter", func(t *testing.T) {
		t.Parallel()
		vars := pkg.ErrorVars("basic.go")
		testkit.True(t, len(vars) > 0, "must find vars in basic.go")
	})

	t.Run("ErrorVars with nonexistent file filter returns empty", func(t *testing.T) {
		t.Parallel()
		vars := pkg.ErrorVars("nonexistent.go")
		testkit.Len(t, vars, 0, "must be empty for nonexistent file")
	})

	t.Run("ConstsOfType returns sorted constants", func(t *testing.T) {
		t.Parallel()
		consts := pkg.ConstsOfType("Status")
		names := make([]string, len(consts))
		for i, c := range consts {
			names[i] = c.Name
		}
		testkit.Equal(t, names, []string{"StatusActive", "StatusClosed", "StatusPending"}, "must be sorted")
	})

	t.Run("ConstsOfType returns empty for nonexistent", func(t *testing.T) {
		t.Parallel()
		testkit.Len(t, pkg.ConstsOfType("Nonexistent"), 0, "must be empty")
	})

	t.Run("Interfaces returns all exported interfaces sorted", func(t *testing.T) {
		t.Parallel()
		ifaces := pkg.Interfaces()
		names := make([]string, len(ifaces))
		for i, iface := range ifaces {
			names[i] = iface.Name
		}
		testkit.Equal(t, names, []string{"Store"}, "must return Store")
	})

	t.Run("Structs returns all exported structs sorted", func(t *testing.T) {
		t.Parallel()
		structs := pkg.Structs()
		names := make([]string, len(structs))
		for i, s := range structs {
			names[i] = s.Name
		}
		want := []string{"Address", "InMemoryStore", "Item", "NotFoundError", "ValidationError", "WrappedError"}
		testkit.Equal(t, names, want, "must return all exported structs")
	})

	t.Run("MethodsOn returns sorted methods on concrete type", func(t *testing.T) {
		t.Parallel()
		concretePkg := loadTestPackage(t, "concrete")
		methods := concretePkg.MethodsOn("Service")
		names := make([]string, len(methods))
		for i, m := range methods {
			names[i] = m.Name
		}
		testkit.Equal(t, names, []string{"Run", "Stop"}, "must be sorted")
	})

	t.Run("MethodsOn extracts doc comment", func(t *testing.T) {
		t.Parallel()
		concretePkg := loadTestPackage(t, "concrete")
		methods := concretePkg.MethodsOn("Service")
		var runDoc string
		for _, m := range methods {
			if m.Name == "Run" {
				runDoc = m.Doc
			}
		}
		testkit.Assert(t, runDoc).Contains("executes", "must extract doc")
	})

	t.Run("MethodsOn returns nil for nonexistent type", func(t *testing.T) {
		t.Parallel()
		concretePkg := loadTestPackage(t, "concrete")
		testkit.True(t, concretePkg.MethodsOn("Nonexistent") == nil, "must return nil")
	})

	t.Run("embedded interface methods are flattened", func(t *testing.T) {
		t.Parallel()
		embeddedPkg := loadTestPackage(t, "embedded")
		rw, err := embeddedPkg.Interface("ReadWriter")
		testkit.NoError(t, err, "must find ReadWriter")
		names := make([]string, len(rw.Methods))
		for i, m := range rw.Methods {
			names[i] = m.Name
		}
		testkit.Equal(t, names, []string{"Read", "Write"}, "must flatten embedded")
	})

	t.Run("deeply embedded interface methods are flattened", func(t *testing.T) {
		t.Parallel()
		embeddedPkg := loadTestPackage(t, "embedded")
		triple, err := embeddedPkg.Interface("TripleReader")
		testkit.NoError(t, err, "must find TripleReader")
		names := make([]string, len(triple.Methods))
		for i, m := range triple.Methods {
			names[i] = m.Name
		}
		testkit.Equal(t, names, []string{"Close", "Read", "Write"}, "must flatten recursively")
	})

	t.Run("ErrorTypes returns custom error types", func(t *testing.T) {
		t.Parallel()
		errorTypes := pkg.ErrorTypes()
		names := make([]string, len(errorTypes))
		for i, et := range errorTypes {
			names[i] = et.Name
		}
		want := []string{"NotFoundError", "ValidationError", "WrappedError"}
		testkit.Equal(t, names, want, "must find error types sorted")
	})

	t.Run("ErrorTypes skips non-error structs", func(t *testing.T) {
		t.Parallel()
		errorTypes := pkg.ErrorTypes()
		for _, et := range errorTypes {
			testkit.True(t, et.Name != "Item", "Item is not an error type")
		}
	})

	t.Run("ErrorTypeHasIs detects Is method", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, pkg.ErrorTypeHasIs("NotFoundError"), "NotFoundError has Is")
		testkit.False(t, pkg.ErrorTypeHasIs("ValidationError"), "ValidationError has no Is")
	})

	t.Run("ErrorTypeHasUnwrap detects Unwrap method", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, pkg.ErrorTypeHasUnwrap("WrappedError"), "WrappedError has Unwrap")
		testkit.False(t, pkg.ErrorTypeHasUnwrap("ValidationError"), "ValidationError has no Unwrap")
		testkit.False(t, pkg.ErrorTypeHasUnwrap("NotFoundError"), "NotFoundError has no Unwrap")
	})

	t.Run("ErrorTypeHasIs returns false for nonexistent type", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, pkg.ErrorTypeHasIs("Nonexistent"), "nonexistent type")
	})

	t.Run("ErrorTypeHasUnwrap returns false for nonexistent type", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, pkg.ErrorTypeHasUnwrap("Nonexistent"), "nonexistent type")
	})

	t.Run("generic interface has type params", func(t *testing.T) {
		t.Parallel()
		genericsPkg := loadTestPackage(t, "generics")
		iface, err := genericsPkg.Interface("Cache")
		testkit.NoError(t, err, "must find Cache")
		testkit.Len(t, iface.TypeParams, 2, "must have 2 type params")
		testkit.Equal(t, iface.TypeParams[0].Name, "K", "first is K")
		testkit.Equal(t, iface.TypeParams[1].Name, "V", "second is V")
	})
}
