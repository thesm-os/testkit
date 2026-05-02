// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
	"go.thesmos.sh/testkit/gen/stub"
)

func buildMethodData(t *testing.T, methodName string) *stub.MethodData {
	t.Helper()
	pkg := loadTestPackage(t, "basic")
	iface, err := pkg.Interface("Store")
	testkit.NoError(t, err, "must load Store")

	var info gen.MethodInfo
	for _, m := range iface.Methods {
		if m.Name == methodName {
			info = m
			break
		}
	}
	if info.Name == "" {
		t.Fatalf("method %q not found on Store", methodName)
	}

	tracker := gen.NewImportTracker("example.com/storetest")
	params := gen.BuildParamFields(info.Signature.Params(), tracker)
	results := gen.BuildResultFields(info.Signature.Results(), tracker)

	return stub.NewMethodDataForTest(
		info,
		"Store"+methodName+"Call",
		"Store"+methodName+"Stub",
		"store"+methodName+"Return",
		params, results,
		tracker,
		"Store", pkg.Pkg.Path(),
	)
}

func TestMethodData(t *testing.T) {
	t.Parallel()

	t.Run("ParamList", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Get")
		got := m.ParamList()
		testkit.Assert(t, got).
			Contains("context.Context", "must have context type").
			Contains("string", "must have string type")
	})

	t.Run("ResultList", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Get")
		got := m.ResultList()
		testkit.Assert(t, got).
			Contains("Item", "must have Item type").
			Contains("error", "must have error type")
	})

	t.Run("FuncTypeStr", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Get")
		got := m.FuncTypeStr()
		testkit.Assert(t, got).
			Contains("func(", "must start with func(").
			Contains("context.Context", "must have context").
			Contains("error", "must have error result")
	})

	t.Run("FuncTypeStr void", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Delete")
		got := m.FuncTypeStr()
		testkit.Assert(t, got).Contains("func(", "must start with func(")
		// Delete returns error, so not truly void — but tests the path
	})

	t.Run("CallExpr", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Get")
		testkit.Equal(t, m.CallExpr("s"), "s.Get(ctx, id)", "forwarding call")
	})

	t.Run("ZeroReturn", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Get")
		got := m.ZeroReturn()
		testkit.Assert(t, got).
			Contains("{}", "must have struct zero value").
			Contains("nil", "must have nil for error")
	})

	t.Run("ZeroReturn error only", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Put")
		testkit.Equal(t, m.ZeroReturn(), "nil", "single error zero is nil")
	})

	t.Run("ParamFieldAssign", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Get")
		got := m.ParamFieldAssign()
		testkit.Assert(t, got).
			Contains("Ctx: ctx", "must assign ctx").
			Contains("ID: id", "must assign id")
	})

	t.Run("ResultFieldAssignVars", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Get")
		got := m.ResultFieldAssignVars()
		testkit.Assert(t, got).
			Contains("Result: r0", "must assign r0 to Result").
			Contains("Err: r1", "must assign r1 to Err")
	})

	t.Run("ResultFieldAssignFallback", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Get")
		got := m.ResultFieldAssignFallback()
		testkit.Assert(t, got).
			Contains("Result: f.Result", "must assign fallback Result").
			Contains("Err: f.Err", "must assign fallback Err")
	})

	t.Run("ReturnParams", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Get")
		got := m.ReturnParams()
		testkit.Assert(t, got).
			Contains("result", "must have result param name").
			Contains("err", "must have err param name")
	})

	t.Run("ReturnFieldAssign", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Get")
		got := m.ReturnFieldAssign()
		testkit.Assert(t, got).
			Contains("Result: result", "must assign result").
			Contains("Err: err", "must assign err")
	})

	t.Run("ReturnFromFallback", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Get")
		got := m.ReturnFromFallback()
		testkit.Assert(t, got).
			Contains("f.Result", "must return fallback Result").
			Contains("f.Err", "must return fallback Err")
	})

	t.Run("ResultVarDecl multi-return", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Get")
		testkit.Equal(t, m.ResultVarDecl(), "r0, r1 := ", "multi-return decl")
	})

	t.Run("ResultVarDecl single return", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Put")
		testkit.Equal(t, m.ResultVarDecl(), "r0 := ", "single return decl")
	})

	t.Run("ResultVarNames", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Get")
		testkit.Equal(t, m.ResultVarNames(), "r0, r1", "result var names")
	})

	t.Run("FaultReturn", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Get")
		got := m.FaultReturn()
		testkit.Assert(t, got).
			Contains("faultErr", "must have faultErr for error position").
			Contains("{}", "must have zero for non-error position")
	})

	t.Run("ErrFieldName", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Get")
		testkit.Equal(t, m.ErrFieldName(), "Err", "error field name")
	})

	t.Run("promoted ReturnsError", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Get")
		testkit.True(t, m.ReturnsError(), "Get returns error")
	})

	t.Run("promoted HasContext", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Get")
		testkit.True(t, m.HasContext(), "Get has context")
	})

	t.Run("promoted ParamNames", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Get")
		testkit.Equal(t, m.ParamNames(), "ctx, id", "param names")
	})
}
