// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package resolver_test

import (
	"go/types"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

func TestSplitQualified(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in       string
		wantPath string
		wantName string
		wantQual bool
		comment  string
	}{
		{"SampleKey", "", "SampleKey", false, "bare ident → local"},
		{"fixtures.SampleKey", "", "fixtures.SampleKey", false, "dotted but no slash → local"},
		{
			"go.thesmos.sh/myproj/fixtures.SampleKey",
			"go.thesmos.sh/myproj/fixtures",
			"SampleKey",
			true,
			"slash in LHS → qualified",
		},
		{"a/b.X", "a/b", "X", true, "minimal qualified form"},
	}
	for _, c := range cases {
		t.Run(c.comment, func(t *testing.T) {
			t.Parallel()
			path, name, qual := resolver.SplitQualified(c.in)
			testkit.Equal(t, path, c.wantPath, "importPath")
			testkit.Equal(t, name, c.wantName, "name")
			testkit.Equal(t, qual, c.wantQual, "qualified")
		})
	}
}

func TestResolved_Render(t *testing.T) {
	t.Parallel()

	t.Run("local: bare name", func(t *testing.T) {
		t.Parallel()
		r := resolver.Resolved{Name: "SampleKey"}
		testkit.Equal(t, r.Render(), "SampleKey", "no alias")
	})
	t.Run("remote: alias.Name", func(t *testing.T) {
		t.Parallel()
		r := resolver.Resolved{Alias: "fixtures", Name: "SampleKey"}
		testkit.Equal(t, r.Render(), "fixtures.SampleKey", "alias prefixed")
	})
}

func TestResolve(t *testing.T) {
	t.Parallel()

	t.Run("local symbol resolves and Obj is non-nil", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadBasicData(t)
		got, err := resolver.Resolve("SampleKey", data, pkg)
		testkit.NoError(t, err, "Resolve")
		testkit.Equal(t, got.Alias, "", "local has no alias")
		testkit.Equal(t, got.Name, "SampleKey", "name preserved")
		testkit.True(t, got.Obj != nil, "Obj resolved")
	})

	t.Run("local symbol missing surfaces error", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadBasicData(t)
		_, err := resolver.Resolve("DoesNotExist", data, pkg)
		testkit.True(t, err != nil, "missing symbol errors")
		testkit.Assert(t, err.Error()).Contains("not found", "diagnostic")
	})

	t.Run("remote: qualified arg loads pkg + registers tracker import", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadBasicData(t)
		const arg = "go.thesmos.sh/testkit/generator/testdata/storage.SampleKey"
		got, err := resolver.Resolve(arg, data, pkg)
		testkit.NoError(t, err, "Resolve")
		testkit.Equal(t, got.Alias, "storage", "alias is the pkg base name")
		testkit.Equal(t, got.Name, "SampleKey", "bare name")
		testkit.Equal(t, got.Render(), "storage.SampleKey", "qualified expression")
	})

	t.Run("remote: missing pkg surfaces load error", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadBasicData(t)
		_, err := resolver.Resolve("go.thesmos.sh/does/not/exist.X", data, pkg)
		testkit.True(t, err != nil, "missing remote pkg errors")
	})

	t.Run("remote: missing symbol surfaces error", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadBasicData(t)
		_, err := resolver.Resolve(
			"go.thesmos.sh/testkit/generator/testdata/storage.DoesNotExist",
			data, pkg)
		testkit.True(t, err != nil, "missing remote symbol errors")
	})

	t.Run("remote: nil Loader surfaces a clear error", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadBasicData(t)
		data.Loader = nil
		_, err := resolver.Resolve(
			"go.thesmos.sh/testkit/generator/testdata/storage.SampleKey",
			data, pkg)
		testkit.True(t, err != nil, "nil Loader rejected")
		testkit.Assert(t, err.Error()).Contains("Loader", "diagnostic mentions Loader")
	})
}

func TestFuncSig_Check(t *testing.T) {
	t.Parallel()

	pkg, _ := loadBasicData(t)
	stringType := returnTypeOf(t, pkg, "SampleKey")
	itemType := returnTypeOf(t, pkg, "SampleItem")

	t.Run("matches func() T", func(t *testing.T) {
		t.Parallel()
		obj := pkg.Pkg.Scope().Lookup("SampleKey")
		err := resolver.FuncSig{Results: []types.Type{stringType}}.Check(obj)
		testkit.NoError(t, err, "func() string accepted")
	})

	t.Run("rejects non-function", func(t *testing.T) {
		t.Parallel()
		// Item is a struct type, not a func.
		obj := pkg.Pkg.Scope().Lookup("Item")
		err := resolver.FuncSig{}.Check(obj)
		testkit.True(t, err != nil, "non-func rejected")
		testkit.Assert(t, err.Error()).Contains("not a function", "diagnostic")
	})

	t.Run("rejects param-count mismatch", func(t *testing.T) {
		t.Parallel()
		// SampleKey is `func() string`; require one param.
		obj := pkg.Pkg.Scope().Lookup("SampleKey")
		err := resolver.FuncSig{
			Params:  []types.Type{stringType},
			Results: []types.Type{stringType},
		}.Check(obj)
		testkit.True(t, err != nil, "param-count mismatch rejected")
		testkit.Assert(t, err.Error()).Contains("expects 1 param", "diagnostic")
	})

	t.Run("rejects result-count mismatch", func(t *testing.T) {
		t.Parallel()
		obj := pkg.Pkg.Scope().Lookup("SampleKey")
		err := resolver.FuncSig{Results: []types.Type{stringType, stringType}}.Check(obj)
		testkit.True(t, err != nil, "result-count mismatch rejected")
		testkit.Assert(t, err.Error()).Contains("expects 2 result", "diagnostic")
	})

	t.Run("rejects type mismatch on results", func(t *testing.T) {
		t.Parallel()
		// SampleKey returns string but we ask for Item.
		obj := pkg.Pkg.Scope().Lookup("SampleKey")
		err := resolver.FuncSig{Results: []types.Type{itemType}}.Check(obj)
		testkit.True(t, err != nil, "type mismatch rejected")
		testkit.Assert(t, err.Error()).Contains("result 0", "diagnostic names result index")
	})

	t.Run("rejects variadic mismatch (false-true)", func(t *testing.T) {
		t.Parallel()
		obj := pkg.Pkg.Scope().Lookup("SampleKey")
		err := resolver.FuncSig{Variadic: true}.Check(obj)
		testkit.True(t, err != nil, "non-variadic rejected when Variadic=true")
		testkit.Assert(t, err.Error()).Contains("must be variadic", "diagnostic")
	})

	t.Run("rejects variadic mismatch (true-false)", func(t *testing.T) {
		t.Parallel()
		// Build a synthetic variadic func: func(...string) and check
		// against FuncSig{Variadic: false}. The inverse-direction
		// branch of the variadic gate.
		fakeFn := types.NewFunc(0, nil, "Probe",
			types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewVar(0, nil, "args",
					types.NewSlice(types.Typ[types.String]))),
				nil, true))
		err := resolver.FuncSig{
			Params: []types.Type{types.NewSlice(types.Typ[types.String])},
		}.Check(fakeFn)
		testkit.True(t, err != nil, "variadic rejected when Variadic=false")
		testkit.Assert(t, err.Error()).Contains("must not be variadic", "diagnostic")
	})

	t.Run("rejects type mismatch on params", func(t *testing.T) {
		t.Parallel()
		// Build a `func(string)` fake func; require `func(int)` →
		// param 0 should diagnose mismatch.
		fakeFn := types.NewFunc(0, nil, "Probe",
			types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewVar(0, nil, "s", types.Typ[types.String])),
				nil, false))
		err := resolver.FuncSig{Params: []types.Type{types.Typ[types.Int]}}.Check(fakeFn)
		testkit.True(t, err != nil, "type mismatch rejected")
		testkit.Assert(t, err.Error()).Contains("param 0", "diagnostic names param index")
	})
}

func TestVarOfType(t *testing.T) {
	t.Parallel()

	t.Run("accepts var assignable to error", func(t *testing.T) {
		t.Parallel()
		pkg, _ := loadBasicData(t)
		obj := pkg.Pkg.Scope().Lookup("ErrNotFound")
		err := resolver.VarOfType(obj, resolver.ErrorType())
		testkit.NoError(t, err, "ErrNotFound is assignable to error")
	})

	t.Run("rejects type (non-var)", func(t *testing.T) {
		t.Parallel()
		pkg, _ := loadBasicData(t)
		obj := pkg.Pkg.Scope().Lookup("Item")
		err := resolver.VarOfType(obj, resolver.ErrorType())
		testkit.True(t, err != nil, "type rejected")
		testkit.Assert(t, err.Error()).Contains("not a variable", "diagnostic")
	})

	t.Run("rejects var with non-assignable type", func(t *testing.T) {
		t.Parallel()
		// Build a synthetic *types.Var with type string. Not
		// assignable to error.
		v := types.NewVar(0, nil, "name", types.Typ[types.String])
		err := resolver.VarOfType(v, resolver.ErrorType())
		testkit.True(t, err != nil, "string-typed var rejected")
		testkit.Assert(t, err.Error()).Contains("not assignable", "diagnostic")
	})
}

func TestErrorType(t *testing.T) {
	t.Parallel()
	got := resolver.ErrorType()
	testkit.True(t, got != nil, "non-nil error type")
	testkit.Equal(t, got.String(), "error", "stringer is `error`")
}

func TestRequireArgs(t *testing.T) {
	t.Parallel()

	t.Run("matching count is OK", func(t *testing.T) {
		t.Parallel()
		err := resolver.RequireArgs(directive.Directive{Args: []string{"a", "b"}}, 2)
		testkit.NoError(t, err, "match")
	})
	t.Run("mismatch surfaces a uniform diagnostic", func(t *testing.T) {
		t.Parallel()
		err := resolver.RequireArgs(directive.Directive{Args: []string{"a"}}, 2)
		testkit.True(t, err != nil, "mismatch errors")
		testkit.Assert(t, err.Error()).
			Contains("expects 2 arg(s)", "want count").
			Contains("got 1", "got count")
	})
}

// loadBasicData builds a (Package, *spec.Data) pair anchored on the
// basic fixture, ready for Resolve calls. spec.Analyze provides the
// shared Loader + Tracker so both the local and qualified branches
// of Resolve work end-to-end.
func loadBasicData(t *testing.T) (*generator.Package, *spec.Data) {
	t.Helper()
	pkg, err := generator.NewLoader().Load("./../../../testdata/basic", "")
	testkit.NoError(t, err, "Load testdata/basic")
	data, err := spec.Analyze(pkg, []string{"Sampler"},
		generator.DefaultConfig(), generator.Options{Output: "samplertest/x.gen.go"})
	testkit.NoError(t, err, "Analyze")
	return pkg, data
}

// returnTypeOf returns the first result type of the named func in
// pkg. Used to pluck out string and Item types for the FuncSig
// fixtures without re-deriving them through go/types boilerplate.
func returnTypeOf(t *testing.T, pkg *generator.Package, funcName string) types.Type {
	t.Helper()
	obj := pkg.Pkg.Scope().Lookup(funcName)
	testkit.True(t, obj != nil, "lookup "+funcName)
	sig := obj.Type().(*types.Signature)
	return sig.Results().At(0).Type()
}
