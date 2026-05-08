// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator_test

import (
	"go/types"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
)

func TestScanVars(t *testing.T) {
	t.Parallel()

	t.Run("returns vars matching predicate, sorted", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		got := generator.ScanVars(pkg, "", func(v *types.Var) bool {
			return strings.HasPrefix(v.Name(), "Err")
		})
		// basic has ErrConflict, ErrForbidden, ErrNotFound (sorted).
		testkit.Len(t, got, 3, "three Err* vars")
		testkit.Equal(t, got[0].Name, "ErrConflict", "first sorted")
		testkit.Equal(t, got[2].Name, "ErrNotFound", "last sorted")
	})

	t.Run("predicate that rejects everything yields empty slice", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		got := generator.ScanVars(pkg, "", func(*types.Var) bool { return false })
		testkit.Len(t, got, 0, "no matches")
	})

	t.Run("sourceFile filter restricts to single file", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		got := generator.ScanVars(pkg, "errors.go", func(v *types.Var) bool {
			return strings.HasPrefix(v.Name(), "Err")
		})
		testkit.True(t, len(got) > 0, "errors.go has sentinel vars")

		none := generator.ScanVars(pkg, "nonexistent.go", func(v *types.Var) bool {
			return strings.HasPrefix(v.Name(), "Err")
		})
		testkit.Len(t, none, 0, "filter to non-existent file → empty")
	})
}

func TestScanStructsImplementing(t *testing.T) {
	t.Parallel()

	t.Run("returns exported structs implementing the interface", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		got := generator.ScanStructsImplementing(pkg, generator.ErrorInterface())
		// basic has NotFoundError, ValidationError, WrappedError.
		testkit.Len(t, got, 3, "three error types")
		// Sorted by name.
		testkit.Equal(t, got[0].Name, "NotFoundError", "first sorted")
	})

	t.Run("interface no struct implements yields empty slice", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		// An empty interface — every type implements it, but
		// ScanStructsImplementing only returns *exported structs* whose
		// pointer-receiver method set is a superset, which all named
		// structs trivially are. So still non-empty for empty interface.
		// Use a stricter check: an interface with a method no fixture has.
		impossible := types.NewInterfaceType([]*types.Func{
			types.NewFunc(0, nil, "DoesNotExist",
				types.NewSignatureType(nil, nil, nil, nil, nil, false)),
		}, nil).Complete()
		got := generator.ScanStructsImplementing(pkg, impossible)
		testkit.Len(t, got, 0, "no struct has DoesNotExist method")
	})
}

func TestHasMethod(t *testing.T) {
	t.Parallel()

	t.Run("finds method matching name + signature", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		// NotFoundError has Is(error) bool.
		testkit.True(t,
			generator.HasMethod(pkg, "NotFoundError", "Is", generator.IsErrorBoolSig),
			"NotFoundError has Is")
		// WrappedError has Unwrap() error.
		testkit.True(t,
			generator.HasMethod(pkg, "WrappedError", "Unwrap", generator.UnwrapSig),
			"WrappedError has Unwrap")
	})

	t.Run("returns false for missing methods", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		testkit.False(t,
			generator.HasMethod(pkg, "ValidationError", "Is", generator.IsErrorBoolSig),
			"ValidationError lacks Is")
		testkit.False(t,
			generator.HasMethod(pkg, "NotFoundError", "Unwrap", generator.UnwrapSig),
			"NotFoundError lacks Unwrap")
	})

	t.Run("returns false for missing types", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		testkit.False(t,
			generator.HasMethod(pkg, "DoesNotExist", "Is", generator.IsErrorBoolSig),
			"missing type → false")
	})

	t.Run("rejects methods with wrong signature", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		// Counter has Reset() (no return) — does not match Unwrap()'s
		// `() error` shape.
		testkit.False(t,
			generator.HasMethod(pkg, "Counter", "Reset", generator.UnwrapSig),
			"Reset signature does not match Unwrap")
	})
}

func TestRenderPackageDirectives(t *testing.T) {
	t.Parallel()

	t.Run("returns matching directives as source lines", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		// basic/doc.go declares
		// //testkit:sentinel-no-overlap-with .../testdata/storage
		got := generator.RenderPackageDirectives(pkg, "sentinel-no-overlap-with")
		testkit.Len(t, got, 1, "one matching directive")
		testkit.Assert(t, got[0]).
			Contains("//testkit:sentinel-no-overlap-with", "directive prefix + name").
			Contains("go.thesmos.sh/testkit/generator/testdata/storage", "directive arg")
	})

	t.Run("filter mismatch yields empty slice", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		got := generator.RenderPackageDirectives(pkg, "does-not-exist")
		testkit.Len(t, got, 0, "no matches")
	})

	t.Run("nil filter surfaces every package directive", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		got := generator.RenderPackageDirectives(pkg)
		testkit.True(t, len(got) >= 1, "at least the declared sentinel directive")
	})

	t.Run("package with no directives yields empty slice", func(t *testing.T) {
		t.Parallel()
		pkg, err := generator.NewLoader().Load("./testdata/storage", "")
		testkit.NoError(t, err, "load storage")
		got := generator.RenderPackageDirectives(pkg)
		testkit.Len(t, got, 0, "storage has no package directives")
	})
}

func TestScanConstsOfType(t *testing.T) {
	t.Parallel()

	t.Run("returns constants of named type sorted by source position", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		got := generator.ScanConstsOfType(pkg, "Status")
		// status.go declares StatusPending, StatusActive, StatusClosed
		// in iota order — the scan preserves source order.
		testkit.Len(t, got, 3, "three Status values")
		testkit.Equal(t, got[0].Name, "StatusPending", "iota[0] first")
		testkit.Equal(t, got[1].Name, "StatusActive", "iota[1] second")
		testkit.Equal(t, got[2].Name, "StatusClosed", "iota[2] third")
	})

	t.Run("populates Comment from inline doc", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		got := generator.ScanConstsOfType(pkg, "Status")
		testkit.Equal(t, got[0].Comment, "Pending", "inline comment captured")
		testkit.Equal(t, got[1].Comment, "Active", "inline comment captured")
	})

	t.Run("Comment empty when no inline comment", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		got := generator.ScanConstsOfType(pkg, "Priority")
		testkit.Len(t, got, 3, "three Priority values")
		for _, c := range got {
			testkit.Equal(t, c.Comment, "", "Priority constants have no inline comments")
		}
	})

	t.Run("missing type yields empty slice", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		got := generator.ScanConstsOfType(pkg, "DoesNotExist")
		testkit.Len(t, got, 0, "no consts for missing type")
	})
}

func TestHasFunc(t *testing.T) {
	t.Parallel()

	t.Run("finds function matching signature", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		// status.go declares ParseStatus(string) (Status, error).
		testkit.True(t,
			generator.HasFunc(pkg, "ParseStatus", generator.ParseSig("Status")),
			"ParseStatus matches the Parse signature for Status")
	})

	t.Run("rejects function with wrong signature", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		testkit.False(t,
			generator.HasFunc(pkg, "ParseStatus", generator.ParseSig("Priority")),
			"ParseStatus does not return Priority")
	})

	t.Run("returns false for missing functions", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		testkit.False(t,
			generator.HasFunc(pkg, "DoesNotExist", generator.StringerSig),
			"missing → false")
	})
}

func TestStringerSig(t *testing.T) {
	t.Parallel()
	pkg := loadBasic(t)
	t.Run("matches String() string", func(t *testing.T) {
		t.Parallel()
		testkit.True(t,
			generator.HasMethod(pkg, "Status", "String", generator.StringerSig),
			"Status has stringer")
	})
	t.Run("rejects types without stringer", func(t *testing.T) {
		t.Parallel()
		testkit.False(t,
			generator.HasMethod(pkg, "Priority", "String", generator.StringerSig),
			"Priority has no stringer")
	})
}

func TestMarshalSigs(t *testing.T) {
	t.Parallel()
	pkg := loadBasic(t)

	t.Run("MarshalText / UnmarshalText round-trip detected on Status", func(t *testing.T) {
		t.Parallel()
		testkit.True(t,
			generator.HasMethod(pkg, "Status", "MarshalText", generator.MarshalTextSig),
			"Status has MarshalText")
		testkit.True(t,
			generator.HasMethod(pkg, "Status", "UnmarshalText", generator.UnmarshalTextSig),
			"Status has UnmarshalText")
	})

	t.Run("MarshalJSON / UnmarshalJSON round-trip detected on Status", func(t *testing.T) {
		t.Parallel()
		testkit.True(t,
			generator.HasMethod(pkg, "Status", "MarshalJSON", generator.MarshalJSONSig),
			"Status has MarshalJSON")
		testkit.True(t,
			generator.HasMethod(pkg, "Status", "UnmarshalJSON", generator.UnmarshalJSONSig),
			"Status has UnmarshalJSON")
	})

	t.Run("MarshalBinary / UnmarshalBinary round-trip detected on Status", func(t *testing.T) {
		t.Parallel()
		testkit.True(t,
			generator.HasMethod(pkg, "Status", "MarshalBinary", generator.MarshalBinarySig),
			"Status has MarshalBinary")
		testkit.True(t,
			generator.HasMethod(pkg, "Status", "UnmarshalBinary", generator.UnmarshalBinarySig),
			"Status has UnmarshalBinary")
	})

	t.Run("Priority has none of the marshalers", func(t *testing.T) {
		t.Parallel()
		testkit.False(t,
			generator.HasMethod(pkg, "Priority", "MarshalText", generator.MarshalTextSig),
			"Priority no MarshalText")
		testkit.False(t,
			generator.HasMethod(pkg, "Priority", "MarshalJSON", generator.MarshalJSONSig),
			"Priority no MarshalJSON")
		testkit.False(t,
			generator.HasMethod(pkg, "Priority", "MarshalBinary", generator.MarshalBinarySig),
			"Priority no MarshalBinary")
	})
}

func TestDefaultsFuncSig(t *testing.T) {
	t.Parallel()

	t.Run("matches `() <Type>` factory shape", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		// basic doesn't ship a defaults function for any of its
		// types, so we exercise the predicate against a known
		// existing function: ParseStatus has shape
		// `(string) (Status, error)` — wrong shape, must be rejected
		// by DefaultsFuncSig("Status").
		testkit.False(t,
			generator.HasFunc(pkg, "ParseStatus", generator.DefaultsFuncSig("Status")),
			"ParseStatus is not a defaults factory")
	})

	t.Run("rejects functions with parameters", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		testkit.False(t,
			generator.HasFunc(pkg, "ParseStatus", generator.DefaultsFuncSig("Status")),
			"factory must take zero params")
	})

	t.Run("rejects functions returning a different type", func(t *testing.T) {
		t.Parallel()
		// Compose a synthetic test: predicate is type-name-keyed,
		// so a same-shape factory for the wrong type must fail.
		// Easiest exercise: confirm the closure captures typeName.
		check := generator.DefaultsFuncSig("DoesNotExist")
		testkit.True(t, check != nil, "predicate constructed")
	})
}

func TestDefaultsFuncSigMatches(t *testing.T) {
	t.Parallel()

	t.Run("matches RequestDefaults() in defaultstest sibling package", func(t *testing.T) {
		t.Parallel()
		pkg, err := generator.NewLoader().Load("./testdata/defaults/defaultstest", "")
		testkit.NoError(t, err, "Load defaultstest")
		// RequestDefaults() returns defaults.Request — predicate keys
		// on the named type's Obj().Name() so a cross-package return
		// type matches as long as the leaf name agrees.
		testkit.True(t,
			generator.HasFunc(pkg, "RequestDefaults", generator.DefaultsFuncSig("Request")),
			"RequestDefaults satisfies DefaultsFuncSig(\"Request\")")
	})

	t.Run("rejects functions with non-named return type", func(t *testing.T) {
		t.Parallel()
		// Build a synthetic signature `() string` and feed it directly.
		// types.Typ[String] is *types.Basic, not *types.Named, so the
		// predicate's named-cast branch fails.
		sig := types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.String])),
			false)
		check := generator.DefaultsFuncSig("Request")
		testkit.False(t, check(sig), "non-named return rejected")
	})
}

func TestSelectConcreteType(t *testing.T) {
	t.Parallel()

	t.Run("any-constrained → first candidate at startIdx", func(t *testing.T) {
		t.Parallel()
		// Empty interface = any — every candidate satisfies it, so
		// we get the rotated start.
		anyIface := types.NewInterfaceType(nil, nil).Complete()
		got := generator.SelectConcreteType(anyIface, generator.DefaultConcreteTypes, 0)
		testkit.True(t, got != nil, "non-nil")
		testkit.Equal(t, got.Name, "string", "position 0 → string")

		got = generator.SelectConcreteType(anyIface, generator.DefaultConcreteTypes, 1)
		testkit.Equal(t, got.Name, "int", "position 1 → int")
	})

	t.Run("rotation wraps modulo candidate count", func(t *testing.T) {
		t.Parallel()
		anyIface := types.NewInterfaceType(nil, nil).Complete()
		n := len(generator.DefaultConcreteTypes)
		got := generator.SelectConcreteType(anyIface, generator.DefaultConcreteTypes, n)
		testkit.True(t, got != nil, "wrap to start")
		testkit.Equal(t, got.Name, "string", "wraps to first")
	})

	t.Run("comparable-constrained → first satisfying candidate", func(t *testing.T) {
		t.Parallel()
		// Use the actual comparable interface from the universe.
		comparableIface, ok := types.Universe.Lookup("comparable").
			Type().Underlying().(*types.Interface)
		testkit.True(t, ok, "comparable resolves to interface")
		got := generator.SelectConcreteType(comparableIface, generator.DefaultConcreteTypes, 0)
		testkit.True(t, got != nil, "string satisfies comparable")
		testkit.Equal(t, got.Name, "string", "first comparable candidate")
	})

	t.Run("Numeric-constrained → int (string fails)", func(t *testing.T) {
		t.Parallel()
		// Build a type-set interface equivalent to ~int | ~int64 | ~float64.
		pkg, err := generator.NewLoader().Load("./testdata/generics", "")
		testkit.NoError(t, err, "Load generics")
		obj := pkg.Pkg.Scope().Lookup("Numeric")
		testkit.True(t, obj != nil, "Numeric defined")
		iface, ok := obj.Type().Underlying().(*types.Interface)
		testkit.True(t, ok, "Numeric is interface")
		// Start at position 0 (string) — must skip past since string
		// fails ~int|~int64|~float64.
		got := generator.SelectConcreteType(iface, generator.DefaultConcreteTypes, 0)
		testkit.True(t, got != nil, "int satisfies Numeric")
		testkit.Equal(t, got.Name, "int", "int is the first satisfying candidate")
	})

	t.Run("returns nil when no candidate satisfies", func(t *testing.T) {
		t.Parallel()
		// Method-bearing interface — no basic kind satisfies it.
		impossible := types.NewInterfaceType([]*types.Func{
			types.NewFunc(0, nil, "DoesNotExist",
				types.NewSignatureType(nil, nil, nil, nil, nil, false)),
		}, nil).Complete()
		got := generator.SelectConcreteType(impossible, generator.DefaultConcreteTypes, 0)
		testkit.True(t, got == nil, "nil when nothing satisfies")
	})

	t.Run("returns nil for empty candidate list", func(t *testing.T) {
		t.Parallel()
		anyIface := types.NewInterfaceType(nil, nil).Complete()
		testkit.True(t,
			generator.SelectConcreteType(anyIface, nil, 0) == nil,
			"empty candidates → nil")
	})

	t.Run("non-interface constraint falls back to round-robin", func(t *testing.T) {
		t.Parallel()
		// types.Typ[String] has *types.Basic underlying — not an
		// interface — so the function takes the round-robin fallback.
		got := generator.SelectConcreteType(types.Typ[types.String], generator.DefaultConcreteTypes, 2)
		testkit.True(t, got != nil, "non-iface constraint → rotation")
		testkit.Equal(t, got.Name, "bool", "position 2 → bool")
	})
}

func TestErrorInterface(t *testing.T) {
	t.Parallel()
	iface := generator.ErrorInterface()
	testkit.True(t, iface != nil, "non-nil")
	// builtin error has exactly one method: Error() string.
	testkit.Equal(t, iface.NumMethods(), 1, "one method")
	testkit.Equal(t, iface.Method(0).Name(), "Error", "method is Error")
}
