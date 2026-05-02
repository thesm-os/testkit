// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import (
	"testing"

	"go.thesmos.sh/testkit"
)

func TestMethodInfo_helpers(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "basic")
	iface, err := pkg.Interface("Store")
	testkit.NoError(t, err, "must load Store")

	// Find Get and Put methods.
	var get, put, del MethodInfo
	for _, m := range iface.Methods {
		switch m.Name {
		case "Get":
			get = m
		case "Put":
			put = m
		case "Delete":
			del = m
		}
	}

	t.Run("HasContext", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, get.HasContext(), "Get must have context")
		testkit.True(t, put.HasContext(), "Put must have context")
	})

	t.Run("ReturnsError", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, get.ReturnsError(), "Get must return error")
		testkit.True(t, put.ReturnsError(), "Put must return error")
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
	})

	t.Run("ParamList", func(t *testing.T) {
		t.Parallel()
		tracker := NewImportTracker("example.com/test")
		got := get.ParamList(tracker)
		testkit.Assert(t, got).
			Contains("context.Context", "must include context type").
			Contains("string", "must include string type")
	})

	t.Run("ParamNames", func(t *testing.T) {
		t.Parallel()
		got := get.ParamNames()
		testkit.Equal(t, got, "ctx, id", "must render parameter names")
	})

	t.Run("ResultList single result", func(t *testing.T) {
		t.Parallel()
		tracker := NewImportTracker("example.com/test")
		got := put.ResultList(tracker)
		testkit.Equal(t, got, "error", "single result without parens")
	})

	t.Run("ResultList multiple results", func(t *testing.T) {
		t.Parallel()
		tracker := NewImportTracker("example.com/test")
		got := get.ResultList(tracker)
		testkit.Assert(t, got).
			Contains("(", "multi-result must have parens").
			Contains("error", "must include error")
	})

	t.Run("CallForward", func(t *testing.T) {
		t.Parallel()
		got := get.CallForward("s")
		testkit.Equal(t, got, "s.Get(ctx, id)", "must render forwarding call")
	})

	t.Run("ZeroResults for Get", func(t *testing.T) {
		t.Parallel()
		tracker := NewImportTracker("example.com/test")
		got := get.ZeroResults(tracker)
		testkit.Assert(t, got).Contains("nil", "must include nil for error")
	})

	t.Run("ZeroResults for Delete returns nil", func(t *testing.T) {
		t.Parallel()
		tracker := NewImportTracker("example.com/test")
		got := del.ZeroResults(tracker)
		testkit.Equal(t, got, "nil", "single error result zero is nil")
	})

	t.Run("FuncType", func(t *testing.T) {
		t.Parallel()
		tracker := NewImportTracker("example.com/test")
		got := put.FuncType(tracker)
		testkit.Assert(t, got).
			Contains("func(", "must start with func(").
			Contains("context.Context", "must include context").
			Contains("error", "must include error result")
	})
}

func TestMethodInfo_no_params(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "concrete")
	methods := pkg.MethodsOn("Service")

	var stop *MethodInfo
	for _, m := range methods {
		if m.Name == "Stop" {
			stop = m
		}
	}

	t.Run("HasContext false for no-param method", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, stop.HasContext(), "Stop has no params")
	})

	t.Run("ReturnsError false for void method", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, stop.ReturnsError(), "Stop returns nothing")
	})

	t.Run("ResultList empty for void method", func(t *testing.T) {
		t.Parallel()
		tracker := NewImportTracker("example.com/test")
		testkit.Equal(t, stop.ResultList(tracker), "", "void must be empty")
	})

	t.Run("ZeroResults empty for void method", func(t *testing.T) {
		t.Parallel()
		tracker := NewImportTracker("example.com/test")
		testkit.Equal(t, stop.ZeroResults(tracker), "", "void must be empty")
	})
}

func TestTypeParams(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "generics")
	iface, err := pkg.Interface("Cache")
	testkit.NoError(t, err, "must load Cache")

	t.Run("TypeParamDecl", func(t *testing.T) {
		t.Parallel()
		tracker := NewImportTracker("example.com/test")
		got := iface.TypeParamDecl(tracker)
		testkit.Assert(t, got).
			Contains("[", "must have brackets").
			Contains("K", "must include K").
			Contains("V", "must include V")
	})

	t.Run("TypeParamArgs", func(t *testing.T) {
		t.Parallel()
		got := iface.TypeParamArgs()
		testkit.Equal(t, got, "[K, V]", "must render type args")
	})

	t.Run("empty for non-generic", func(t *testing.T) {
		t.Parallel()
		basicPkg := loadTestPackage(t, "basic")
		store, err := basicPkg.Interface("Store")
		testkit.NoError(t, err, "must load Store")
		tracker := NewImportTracker("example.com/test")
		testkit.Equal(t, store.TypeParamDecl(tracker), "", "non-generic must be empty")
		testkit.Equal(t, store.TypeParamArgs(), "", "non-generic must be empty")
	})
}
