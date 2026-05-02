// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen_test

import (
	"go/types"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
)

func TestMethodInfo(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "basic")
	iface, err := pkg.Interface("Store")
	testkit.NoError(t, err, "must load Store")

	var get, put, del, find gen.MethodInfo
	for _, m := range iface.Methods {
		switch m.Name {
		case "Get":
			get = m
		case "Put":
			put = m
		case "Delete":
			del = m
		case "Find":
			find = m
		}
	}

	t.Run("HasContext on method with context", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, get.HasContext(), "Get must have context")
	})

	t.Run("ReturnsError on error-returning method", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, get.ReturnsError(), "Get must return error")
	})

	t.Run("NumParams", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, get.NumParams(), 2, "Get has ctx and id")
		testkit.Equal(t, put.NumParams(), 2, "Put has ctx and item")
	})

	t.Run("NumResults", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, get.NumResults(), 2, "Get returns Item and error")
		testkit.Equal(t, put.NumResults(), 1, "Put returns error")
	})

	t.Run("IsVariadic", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, get.IsVariadic(), "Get is not variadic")
		testkit.True(t, find.IsVariadic(), "Find is variadic")
	})

	t.Run("ParamList", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/test")
		got := get.ParamList(tracker)
		testkit.Assert(t, got).
			Contains("context.Context", "must include context type").
			Contains("string", "must include string type")
	})

	t.Run("ParamNames", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, get.ParamNames(), "ctx, id", "must render parameter names")
	})

	t.Run("ResultList single result", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/test")
		testkit.Equal(t, put.ResultList(tracker), "error", "single result without parens")
	})

	t.Run("ResultList multiple results", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/test")
		got := get.ResultList(tracker)
		testkit.Assert(t, got).
			Contains("(", "multi-result must have parens").
			Contains("error", "must include error")
	})

	t.Run("CallForward", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, get.CallForward("s"), "s.Get(ctx, id)", "must render forwarding call")
	})

	t.Run("ZeroResults for multi-return", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/test")
		testkit.Assert(t, get.ZeroResults(tracker)).Contains("nil", "must include nil for error")
	})

	t.Run("ZeroResults for error-only", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/test")
		testkit.Equal(t, del.ZeroResults(tracker), "nil", "single error result zero is nil")
	})

	t.Run("FuncType", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/test")
		got := put.FuncType(tracker)
		testkit.Assert(t, got).
			Contains("func(", "must start with func(").
			Contains("context.Context", "must include context").
			Contains("error", "must include error result")
	})

	t.Run("variadic ParamList uses ellipsis", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/test")
		got := find.ParamList(tracker)
		testkit.Assert(t, got).Contains("...string", "variadic must use ellipsis")
	})

	t.Run("variadic FuncType uses ellipsis", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/test")
		got := find.FuncType(tracker)
		testkit.Assert(t, got).
			Contains("...string", "variadic must use ellipsis in func type").
			Contains("func(", "must start with func(")
	})

	t.Run("no-param method HasContext is false", func(t *testing.T) {
		t.Parallel()
		concretePkg := loadTestPackage(t, "concrete")
		methods := concretePkg.MethodsOn("Service")
		var stop *gen.MethodInfo
		for _, m := range methods {
			if m.Name == "Stop" {
				stop = m
			}
		}
		testkit.False(t, stop.HasContext(), "Stop has no params")
		testkit.False(t, stop.ReturnsError(), "Stop returns nothing")
	})

	t.Run("void method ResultList and ZeroResults are empty", func(t *testing.T) {
		t.Parallel()
		concretePkg := loadTestPackage(t, "concrete")
		methods := concretePkg.MethodsOn("Service")
		var stop *gen.MethodInfo
		for _, m := range methods {
			if m.Name == "Stop" {
				stop = m
			}
		}
		tracker := gen.NewImportTracker("example.com/test")
		testkit.Equal(t, stop.ResultList(tracker), "", "void ResultList must be empty")
		testkit.Equal(t, stop.ZeroResults(tracker), "", "void ZeroResults must be empty")
	})
}

func TestInterfaceInfoTypeParams(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "generics")
	iface, err := pkg.Interface("Cache")
	testkit.NoError(t, err, "must load Cache")

	t.Run("TypeParamDecl renders declaration", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/test")
		got := iface.TypeParamDecl(tracker)
		testkit.Assert(t, got).
			Contains("[", "must have brackets").
			Contains("K", "must include K").
			Contains("V", "must include V")
	})

	t.Run("TypeParamArgs renders args", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, iface.TypeParamArgs(), "[K, V]", "must render type args")
	})

	t.Run("non-generic returns empty", func(t *testing.T) {
		t.Parallel()
		basicPkg := loadTestPackage(t, "basic")
		store, err := basicPkg.Interface("Store")
		testkit.NoError(t, err, "must load Store")
		tracker := gen.NewImportTracker("example.com/test")
		testkit.Equal(t, store.TypeParamDecl(tracker), "", "non-generic must be empty")
		testkit.Equal(t, store.TypeParamArgs(), "", "non-generic must be empty")
	})
}

func TestZeroValueOf(t *testing.T) {
	t.Parallel()
	tracker := gen.NewImportTracker("example.com/test")

	t.Run("string", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.ZeroValueOf(types.Typ[types.String], tracker), `""`, "string zero")
	})

	t.Run("int", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.ZeroValueOf(types.Typ[types.Int], tracker), "0", "int zero")
	})

	t.Run("bool", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.ZeroValueOf(types.Typ[types.Bool], tracker), "false", "bool zero")
	})

	t.Run("pointer is nil", func(t *testing.T) {
		t.Parallel()
		ptr := types.NewPointer(types.Typ[types.String])
		testkit.Equal(t, gen.ZeroValueOf(ptr, tracker), "nil", "pointer zero")
	})

	t.Run("slice is nil", func(t *testing.T) {
		t.Parallel()
		sl := types.NewSlice(types.Typ[types.String])
		testkit.Equal(t, gen.ZeroValueOf(sl, tracker), "nil", "slice zero")
	})

	t.Run("map is nil", func(t *testing.T) {
		t.Parallel()
		mp := types.NewMap(types.Typ[types.String], types.Typ[types.Int])
		testkit.Equal(t, gen.ZeroValueOf(mp, tracker), "nil", "map zero")
	})

	t.Run("chan is nil", func(t *testing.T) {
		t.Parallel()
		ch := types.NewChan(types.SendRecv, types.Typ[types.String])
		testkit.Equal(t, gen.ZeroValueOf(ch, tracker), "nil", "chan zero")
	})

	t.Run("func is nil", func(t *testing.T) {
		t.Parallel()
		sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
		testkit.Equal(t, gen.ZeroValueOf(sig, tracker), "nil", "func zero")
	})

	t.Run("interface is nil", func(t *testing.T) {
		t.Parallel()
		iface := types.NewInterfaceType(nil, nil)
		iface.Complete()
		testkit.Equal(t, gen.ZeroValueOf(iface, tracker), "nil", "interface zero")
	})

	t.Run("named struct uses qualified suffix", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		iface, err := pkg.Interface("Store")
		testkit.NoError(t, err, "must load Store")
		var get gen.MethodInfo
		for _, m := range iface.Methods {
			if m.Name == "Get" {
				get = m
			}
		}
		resultType := get.Signature.Results().At(0).Type()
		tr := gen.NewImportTracker("example.com/other")
		got := gen.ZeroValueOf(resultType, tr)
		testkit.Assert(t, got).Contains("{}", "named struct zero uses suffix")
	})

	t.Run("array uses suffix", func(t *testing.T) {
		t.Parallel()
		arr := types.NewArray(types.Typ[types.Int], 3)
		got := gen.ZeroValueOf(arr, tracker)
		testkit.Assert(t, got).Contains("{}", "array zero uses suffix")
	})

	t.Run("named interface is nil", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		obj := pkg.Pkg.Scope().Lookup("Store")
		got := gen.ZeroValueOf(obj.Type(), tracker)
		testkit.Equal(t, got, "nil", "named interface zero is nil")
	})
}

func TestStructInfoTypeParams(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "generics")

	t.Run("generic struct has type params", func(t *testing.T) {
		t.Parallel()
		s, err := pkg.Struct("Pair")
		testkit.NoError(t, err, "must load Pair")
		testkit.Len(t, s.TypeParams, 2, "must have 2 type params")
		testkit.Equal(t, s.TypeParams[0].Name, "A", "first is A")
		testkit.Equal(t, s.TypeParams[1].Name, "B", "second is B")
	})

	t.Run("TypeParamDecl renders declaration", func(t *testing.T) {
		t.Parallel()
		s, err := pkg.Struct("Pair")
		testkit.NoError(t, err, "must load Pair")
		tracker := gen.NewImportTracker("example.com/test")
		got := s.TypeParamDecl(tracker)
		testkit.Assert(t, got).
			Contains("[", "must have brackets").
			Contains("A", "must include A").
			Contains("B", "must include B")
	})

	t.Run("TypeParamArgs renders args", func(t *testing.T) {
		t.Parallel()
		s, err := pkg.Struct("Pair")
		testkit.NoError(t, err, "must load Pair")
		testkit.Equal(t, s.TypeParamArgs(), "[A, B]", "must render type args")
	})

	t.Run("non-generic struct returns empty", func(t *testing.T) {
		t.Parallel()
		basicPkg := loadTestPackage(t, "basic")
		s, err := basicPkg.Struct("Item")
		testkit.NoError(t, err, "must load Item")
		tracker := gen.NewImportTracker("example.com/test")
		testkit.Equal(t, s.TypeParamDecl(tracker), "", "non-generic must be empty")
		testkit.Equal(t, s.TypeParamArgs(), "", "non-generic must be empty")
	})
}

func TestIsContextType(t *testing.T) {
	t.Parallel()

	t.Run("context param detected", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		iface, err := pkg.Interface("Store")
		testkit.NoError(t, err, "must load Store")
		var get gen.MethodInfo
		for _, m := range iface.Methods {
			if m.Name == "Get" {
				get = m
			}
		}
		testkit.True(t, gen.IsContextType(get.Signature.Params().At(0).Type()), "first param must be context")
	})

	t.Run("string is not context", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, gen.IsContextType(types.Typ[types.String]), "string is not context")
	})
}

func TestIsErrorType(t *testing.T) {
	t.Parallel()

	t.Run("error return detected", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		iface, err := pkg.Interface("Store")
		testkit.NoError(t, err, "must load Store")
		var get gen.MethodInfo
		for _, m := range iface.Methods {
			if m.Name == "Get" {
				get = m
			}
		}
		testkit.True(t, gen.IsErrorType(get.Signature.Results().At(1).Type()), "last result must be error")
	})

	t.Run("string is not error", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, gen.IsErrorType(types.Typ[types.String]), "string is not error")
	})
}
