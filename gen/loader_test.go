// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import (
	"path/filepath"
	"runtime"
	"testing"

	"go.thesmos.sh/testkit"
)

func testdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	return filepath.Join(filepath.Dir(file), "testdata")
}

func loadTestPackage(t *testing.T, subdir string) *Package {
	t.Helper()
	loader := NewLoader()
	dir := filepath.Join(testdataDir(t), subdir)
	pkg, err := loader.Load(".", dir)
	testkit.NoError(t, err, "must load package")
	return pkg
}

func TestLoader(t *testing.T) {
	t.Parallel()

	t.Run("caches by import path", func(t *testing.T) {
		t.Parallel()
		loader := NewLoader()
		dir := filepath.Join(testdataDir(t), "basic")
		p1, err := loader.Load(".", dir)
		testkit.NoError(t, err, "first load")
		p2, err := loader.Load(".", dir)
		testkit.NoError(t, err, "second load")
		testkit.True(t, p1 == p2, "must return same pointer from cache")
	})

	t.Run("returns error for nonexistent package", func(t *testing.T) {
		t.Parallel()
		loader := NewLoader()
		_, err := loader.Load(".", "/nonexistent/path/to/package")
		testkit.Error(t, err, "must fail for nonexistent package")
	})
}

func TestPackage_Interface(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "basic")

	t.Run("loads Store interface", func(t *testing.T) {
		t.Parallel()
		iface, err := pkg.Interface("Store")
		testkit.NoError(t, err, "must find Store")
		testkit.Equal(t, iface.Name, "Store", "name must match")
		testkit.Len(t, iface.Methods, 3, "must have 3 methods")
	})

	t.Run("methods are sorted by name", func(t *testing.T) {
		t.Parallel()
		iface, err := pkg.Interface("Store")
		testkit.NoError(t, err, "must find Store")
		names := make([]string, len(iface.Methods))
		for i, m := range iface.Methods {
			names[i] = m.Name
		}
		testkit.Equal(t, names, []string{"Delete", "Get", "Put"}, "must be sorted")
	})

	t.Run("method doc comments are extracted", func(t *testing.T) {
		t.Parallel()
		iface, err := pkg.Interface("Store")
		testkit.NoError(t, err, "must find Store")
		var getDoc string
		for _, m := range iface.Methods {
			if m.Name == "Get" {
				getDoc = m.Doc
			}
		}
		testkit.Assert(t, getDoc).Contains("retrieves an item", "must extract doc comment")
	})

	t.Run("type doc comment is extracted", func(t *testing.T) {
		t.Parallel()
		iface, err := pkg.Interface("Store")
		testkit.NoError(t, err, "must find Store")
		testkit.Assert(t, iface.Doc).Contains("manages items", "must extract type doc")
	})

	t.Run("not found returns error", func(t *testing.T) {
		t.Parallel()
		_, err := pkg.Interface("Nonexistent")
		testkit.Error(t, err, "must fail for missing type")
	})

	t.Run("struct type returns error", func(t *testing.T) {
		t.Parallel()
		_, err := pkg.Interface("Item")
		testkit.Error(t, err, "must fail for struct, not interface")
	})

	t.Run("no type params for non-generic", func(t *testing.T) {
		t.Parallel()
		iface, err := pkg.Interface("Store")
		testkit.NoError(t, err, "must find Store")
		testkit.Len(t, iface.TypeParams, 0, "non-generic must have no type params")
	})
}

func TestPackage_Struct(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "basic")

	t.Run("loads Item struct", func(t *testing.T) {
		t.Parallel()
		s, err := pkg.Struct("Item")
		testkit.NoError(t, err, "must find Item")
		testkit.Equal(t, s.Name, "Item", "name must match")
		testkit.Len(t, s.Fields, 4, "must have 4 fields including unexported")
	})

	t.Run("field info includes export status", func(t *testing.T) {
		t.Parallel()
		s, err := pkg.Struct("Item")
		testkit.NoError(t, err, "must find Item")
		var exported, unexported int
		for _, f := range s.Fields {
			if f.Exported {
				exported++
			} else {
				unexported++
			}
		}
		testkit.Equal(t, exported, 3, "must have 3 exported fields")
		testkit.Equal(t, unexported, 1, "must have 1 unexported field")
	})

	t.Run("not found returns error", func(t *testing.T) {
		t.Parallel()
		_, err := pkg.Struct("Nonexistent")
		testkit.Error(t, err, "must fail for missing type")
	})

	t.Run("interface type returns error", func(t *testing.T) {
		t.Parallel()
		_, err := pkg.Struct("Store")
		testkit.Error(t, err, "must fail for interface, not struct")
	})
}

func TestPackage_Var(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "basic")

	t.Run("loads ErrNotFound", func(t *testing.T) {
		t.Parallel()
		v, err := pkg.Var("ErrNotFound")
		testkit.NoError(t, err, "must find ErrNotFound")
		testkit.Equal(t, v.Name, "ErrNotFound", "name must match")
	})

	t.Run("doc comment is extracted", func(t *testing.T) {
		t.Parallel()
		v, err := pkg.Var("ErrNotFound")
		testkit.NoError(t, err, "must find ErrNotFound")
		testkit.Assert(t, v.Doc).Contains("not found", "must extract doc")
	})

	t.Run("not found returns error", func(t *testing.T) {
		t.Parallel()
		_, err := pkg.Var("Nonexistent")
		testkit.Error(t, err, "must fail for missing var")
	})
}

func TestPackage_Const(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "basic")

	t.Run("loads StatusPending", func(t *testing.T) {
		t.Parallel()
		c, err := pkg.Const("StatusPending")
		testkit.NoError(t, err, "must find StatusPending")
		testkit.Equal(t, c.Name, "StatusPending", "name must match")
	})

	t.Run("not found returns error", func(t *testing.T) {
		t.Parallel()
		_, err := pkg.Const("Nonexistent")
		testkit.Error(t, err, "must fail for missing const")
	})
}

func TestPackage_ErrorVars(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "basic")

	t.Run("returns only exported Err* vars sorted", func(t *testing.T) {
		t.Parallel()
		vars := pkg.ErrorVars()
		names := make([]string, len(vars))
		for i, v := range vars {
			names[i] = v.Name
		}
		testkit.Equal(t, names, []string{"ErrConflict", "ErrNotFound"}, "must return sorted Err* vars")
	})
}

func TestPackage_ConstsOfType(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "basic")

	t.Run("returns Status constants sorted", func(t *testing.T) {
		t.Parallel()
		consts := pkg.ConstsOfType("Status")
		names := make([]string, len(consts))
		for i, c := range consts {
			names[i] = c.Name
		}
		testkit.Equal(t, names, []string{"StatusActive", "StatusClosed", "StatusPending"}, "must return sorted")
	})

	t.Run("returns empty for nonexistent type", func(t *testing.T) {
		t.Parallel()
		consts := pkg.ConstsOfType("Nonexistent")
		testkit.Len(t, consts, 0, "must be empty")
	})
}

func TestPackage_Interfaces_scan(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "basic")

	t.Run("returns all exported interfaces sorted", func(t *testing.T) {
		t.Parallel()
		ifaces := pkg.Interfaces()
		names := make([]string, len(ifaces))
		for i, iface := range ifaces {
			names[i] = iface.Name
		}
		testkit.Equal(t, names, []string{"Store"}, "must return Store")
	})
}

func TestPackage_Structs_scan(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "basic")

	t.Run("returns all exported structs sorted", func(t *testing.T) {
		t.Parallel()
		structs := pkg.Structs()
		names := make([]string, len(structs))
		for i, s := range structs {
			names[i] = s.Name
		}
		testkit.Equal(t, names, []string{"Item"}, "must return Item")
	})
}

func TestPackage_Generics(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "generics")

	t.Run("loads generic interface with type params", func(t *testing.T) {
		t.Parallel()
		iface, err := pkg.Interface("Cache")
		testkit.NoError(t, err, "must find Cache")
		testkit.Len(t, iface.TypeParams, 2, "must have 2 type params")
		testkit.Equal(t, iface.TypeParams[0].Name, "K", "first param must be K")
		testkit.Equal(t, iface.TypeParams[1].Name, "V", "second param must be V")
	})
}

func TestPackage_MethodsOn(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "concrete")

	t.Run("returns methods on concrete type sorted", func(t *testing.T) {
		t.Parallel()
		methods := pkg.MethodsOn("Service")
		names := make([]string, len(methods))
		for i, m := range methods {
			names[i] = m.Name
		}
		testkit.Equal(t, names, []string{"Run", "Stop"}, "must return sorted methods")
	})

	t.Run("method doc is extracted", func(t *testing.T) {
		t.Parallel()
		methods := pkg.MethodsOn("Service")
		var runDoc string
		for _, m := range methods {
			if m.Name == "Run" {
				runDoc = m.Doc
			}
		}
		testkit.Assert(t, runDoc).Contains("executes", "must extract doc")
	})

	t.Run("nonexistent type returns nil", func(t *testing.T) {
		t.Parallel()
		methods := pkg.MethodsOn("Nonexistent")
		testkit.True(t, methods == nil, "must return nil for missing type")
	})
}

func TestPackage_EmbeddedFlattening(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "embedded")

	t.Run("ReadWriter has Read and Write", func(t *testing.T) {
		t.Parallel()
		iface, err := pkg.Interface("ReadWriter")
		testkit.NoError(t, err, "must find ReadWriter")
		names := make([]string, len(iface.Methods))
		for i, m := range iface.Methods {
			names[i] = m.Name
		}
		testkit.Equal(t, names, []string{"Read", "Write"}, "must flatten embedded methods")
	})

	t.Run("TripleReader has Close Read Write", func(t *testing.T) {
		t.Parallel()
		iface, err := pkg.Interface("TripleReader")
		testkit.NoError(t, err, "must find TripleReader")
		names := make([]string, len(iface.Methods))
		for i, m := range iface.Methods {
			names[i] = m.Name
		}
		testkit.Equal(t, names, []string{"Close", "Read", "Write"}, "must flatten recursively")
	})
}
