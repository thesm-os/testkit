// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator_test

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
)

// loadIface parses src and returns the named interface plus a fresh
// import tracker scoped to "p".
func loadIface(t *testing.T, src, name string) (*types.Interface, *generator.ImportTracker) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "src.go", src, parser.ParseComments)
	testkit.NoError(t, err, "parse")
	conf := types.Config{Importer: importer.Default()}
	info := &types.Info{Defs: make(map[*ast.Ident]types.Object)}
	pkg, err := conf.Check("p", fset, []*ast.File{f}, info)
	testkit.NoError(t, err, "typecheck")
	obj := pkg.Scope().Lookup(name)
	testkit.True(t, obj != nil, "interface "+name+" defined")
	return obj.Type().Underlying().(*types.Interface), generator.NewImportTracker("p")
}

// methodSig returns the signature of the named method on iface.
// *types.Interface.Method(i) returns methods in alphabetical order, so
// callers should look up by name rather than index.
func methodSig(t *testing.T, iface *types.Interface, name string) *types.Signature {
	t.Helper()
	for m := range iface.Methods() {
		if m.Name() == name {
			return m.Type().(*types.Signature)
		}
	}
	t.Fatalf("method %q not found", name)
	return nil
}

func TestMethodHelpers(t *testing.T) {
	t.Parallel()

	t.Run("IsContextType detects context.Context first param", func(t *testing.T) {
		t.Parallel()
		const src = `
package p
import "context"
type I interface {
	A(ctx context.Context)
	B(s string)
}
`
		iface, _ := loadIface(t, src, "I")
		a := methodSig(t, iface, "A")
		b := methodSig(t, iface, "B")
		testkit.True(t, generator.IsContextType(a.Params().At(0).Type()), "A's ctx param")
		testkit.False(t, generator.IsContextType(b.Params().At(0).Type()), "B's string param")
	})

	t.Run("IsErrorType distinguishes builtin error", func(t *testing.T) {
		t.Parallel()
		const src = `
package p
type E interface { error }
type I interface {
	A() error
	B() E
	C() string
}
`
		iface, _ := loadIface(t, src, "I")
		cases := map[string]bool{"A": true, "B": false, "C": false}
		for name, want := range cases {
			sig := methodSig(t, iface, name)
			got := generator.IsErrorType(sig.Results().At(0).Type())
			testkit.Equal(t, got, want, "IsErrorType("+name+")")
		}
	})

	t.Run("AnalyzeIterReturn detects iter.Seq and iter.Seq2", func(t *testing.T) {
		t.Parallel()
		const src = `
package p
import "iter"
type I interface {
	Stream() iter.Seq[int]
	StreamErr() iter.Seq2[int, error]
	NotStream() int
}
`
		iface, tracker := loadIface(t, src, "I")
		cases := map[string]struct{ isSeq, isSeq2 bool }{
			"Stream":    {true, false},
			"StreamErr": {false, true},
			"NotStream": {false, false},
		}
		for name, want := range cases {
			sig := methodSig(t, iface, name)
			info := generator.AnalyzeIterReturn(sig.Results().At(0).Type(), tracker)
			testkit.Equal(t, info.IsSeq, want.isSeq, "IsSeq("+name+")")
			testkit.Equal(t, info.IsSeq2, want.isSeq2, "IsSeq2("+name+")")
		}
	})

	t.Run("ZeroValueOf produces correct literal per kind", func(t *testing.T) {
		t.Parallel()
		const src = `
package p
type Item struct{ V int }
type I interface {
	S() string
	I() int
	B() bool
	P() *Item
	Sl() []int
	M() map[string]int
	C() chan int
	St() Item
}
`
		iface, tracker := loadIface(t, src, "I")
		cases := map[string]string{
			"S":  `""`,
			"I":  "0",
			"B":  "false",
			"P":  "nil",
			"Sl": "nil",
			"M":  "nil",
			"C":  "nil",
			"St": "Item{}",
		}
		for name, want := range cases {
			sig := methodSig(t, iface, name)
			testkit.Equal(t, generator.ZeroValueOf(sig.Results().At(0).Type(), tracker), want, "ZeroValueOf("+name+")")
		}
	})

	t.Run("MethodInfo helpers cover happy path", func(t *testing.T) {
		t.Parallel()
		const src = `
package p
import "context"
type I interface {
	Get(ctx context.Context, key string) (string, error)
}
`
		iface, tracker := loadIface(t, src, "I")
		mi := generator.MethodInfo{Name: "Get", Signature: methodSig(t, iface, "Get")}

		testkit.True(t, mi.HasContext(), "HasContext")
		testkit.True(t, mi.ReturnsError(), "ReturnsError")
		testkit.Equal(t, mi.NumParams(), 2, "NumParams")
		testkit.Equal(t, mi.NumResults(), 2, "NumResults")
		testkit.False(t, mi.IsVariadic(), "IsVariadic")
		testkit.Equal(t, mi.ParamList(tracker), "ctx context.Context, key string", "ParamList")
		testkit.Equal(t, mi.ParamNames(), "ctx, key", "ParamNames")
		testkit.Equal(t, mi.CallForward("recv"), "recv.Get(ctx, key)", "CallForward")
		testkit.Equal(t, mi.ZeroResults(tracker), `"", nil`, "ZeroResults")
	})

	t.Run("MethodInfo handles variadic parameters", func(t *testing.T) {
		t.Parallel()
		const src = `
package p
import "context"
type I interface {
	Find(ctx context.Context, ids ...string) ([]string, error)
}
`
		iface, tracker := loadIface(t, src, "I")
		mi := generator.MethodInfo{Name: "Find", Signature: methodSig(t, iface, "Find")}
		testkit.True(t, mi.IsVariadic(), "IsVariadic")
		testkit.Equal(t, mi.ParamNames(), "ctx, ids...", "spread suffix on variadic")
		testkit.Equal(t, mi.ParamList(tracker), "ctx context.Context, ids ...string", "variadic ParamList")
	})

	t.Run("ParamName synthesizes pN names", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, generator.ParamName(0), "p0", "ParamName(0)")
		testkit.Equal(t, generator.ParamName(7), "p7", "ParamName(7)")
	})

	t.Run("ResultList renders multi-return tuples with parens", func(t *testing.T) {
		t.Parallel()
		const src = `package p
type I interface {
	Single() string
	Multi() (int, error)
	Void()
}
`
		iface, tracker := loadIface(t, src, "I")
		single := generator.MethodInfo{Signature: methodSig(t, iface, "Single")}
		multi := generator.MethodInfo{Signature: methodSig(t, iface, "Multi")}
		void := generator.MethodInfo{Signature: methodSig(t, iface, "Void")}

		testkit.Equal(t, single.ResultList(tracker), "string", "single result no parens")
		testkit.Equal(t, multi.ResultList(tracker), "(int, error)", "multi result wraps in parens")
		testkit.Equal(t, void.ResultList(tracker), "", "void result is empty")
	})

	t.Run("FuncType renders the method signature without name", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
	Get(ctx context.Context, k string) (int, error)
	Void()
	Single() string
}
`
		iface, tracker := loadIface(t, src, "I")
		get := generator.MethodInfo{Signature: methodSig(t, iface, "Get")}
		void := generator.MethodInfo{Signature: methodSig(t, iface, "Void")}
		single := generator.MethodInfo{Signature: methodSig(t, iface, "Single")}

		testkit.Equal(t, get.FuncType(tracker), "func(context.Context, string) (int, error)", "FuncType")
		testkit.Equal(t, void.FuncType(tracker), "func()", "void FuncType")
		testkit.Equal(t, single.FuncType(tracker), "func() string", "single-result FuncType")
	})

	t.Run("FuncType handles variadic last param", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface { F(ctx context.Context, ids ...string) error }
`
		iface, tracker := loadIface(t, src, "I")
		mi := generator.MethodInfo{Signature: methodSig(t, iface, "F")}
		testkit.Equal(t, mi.FuncType(tracker), "func(context.Context, ...string) error", "variadic FuncType")
	})

	t.Run("HasContext / ReturnsError edge cases", func(t *testing.T) {
		t.Parallel()
		const src = `package p
type I interface {
	NoParams()
	NoResults(s string)
}
`
		iface, _ := loadIface(t, src, "I")
		empty := generator.MethodInfo{Signature: methodSig(t, iface, "NoParams")}
		testkit.False(t, empty.HasContext(), "no params → no context")
		testkit.False(t, empty.ReturnsError(), "no results → no error")

		noResult := generator.MethodInfo{Signature: methodSig(t, iface, "NoResults")}
		testkit.False(t, noResult.HasContext(), "non-ctx first param")
	})

	t.Run("ZeroValueOf handles type parameters and complex builtins", func(t *testing.T) {
		t.Parallel()
		const src = `package p
type Cache[K comparable, V any] interface { Get(k K) V }
type I interface {
	Bytes() []byte
	Inter() interface{}
}
`
		iface, tracker := loadIface(t, src, "I")
		bytesSig := methodSig(t, iface, "Bytes")
		interSig := methodSig(t, iface, "Inter")
		testkit.Equal(t, generator.ZeroValueOf(bytesSig.Results().At(0).Type(), tracker), "nil", "[]byte zero")
		testkit.Equal(t, generator.ZeroValueOf(interSig.Results().At(0).Type(), tracker), "nil", "interface{} zero")
	})

	t.Run("TypeStr without tracker uses default qualifier", func(t *testing.T) {
		t.Parallel()
		const src = `package p
type I interface { F() string }
`
		iface, _ := loadIface(t, src, "I")
		sig := methodSig(t, iface, "F")
		testkit.Equal(t, generator.TypeStr(sig.Results().At(0).Type(), nil), "string", "no tracker fallback")
	})
}

func TestQualifyType(t *testing.T) {
	t.Parallel()

	t.Run("prefixes typeName with qualifier", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, generator.QualifyType("store", "Item"), "store.Item", "qualified")
	})
	t.Run("returns typeName unchanged for empty qualifier", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, generator.QualifyType("", "Item"), "Item", "no qualifier")
	})
}

func TestTypeParamRendering(t *testing.T) {
	t.Parallel()

	loadGenerics := func(t *testing.T) *generator.Package {
		t.Helper()
		pkg, err := generator.NewLoader().Load("./testdata/generics", "")
		testkit.NoError(t, err, "Load testdata/generics")
		return pkg
	}

	t.Run("StructInfo emits TypeParamDecl + TypeParamArgs for generic struct", func(t *testing.T) {
		t.Parallel()
		pkg := loadGenerics(t)
		s, err := pkg.Struct("Container")
		testkit.NoError(t, err, "Struct Container")
		tracker := generator.NewImportTracker(pkg.Path())
		testkit.Equal(t, s.TypeParamDecl(tracker), "[T any]", "single-param decl")
		testkit.Equal(t, s.TypeParamArgs(), "[T]", "single-param args")
	})

	t.Run("StructInfo handles multi-parameter generic", func(t *testing.T) {
		t.Parallel()
		pkg := loadGenerics(t)
		s, err := pkg.Struct("Pair")
		testkit.NoError(t, err, "Struct Pair")
		tracker := generator.NewImportTracker(pkg.Path())
		// go/types normalizes shared constraints into per-param form.
		testkit.Equal(t, s.TypeParamDecl(tracker), "[A any, B any]", "two-param decl")
		testkit.Equal(t, s.TypeParamArgs(), "[A, B]", "two-param args")
	})

	t.Run("StructInfo renders constrained parameter", func(t *testing.T) {
		t.Parallel()
		pkg := loadGenerics(t)
		s, err := pkg.Struct("Lookup")
		testkit.NoError(t, err, "Struct Lookup")
		tracker := generator.NewImportTracker(pkg.Path())
		testkit.Equal(t, s.TypeParamDecl(tracker), "[K comparable, V any]", "comparable constraint")
		testkit.Equal(t, s.TypeParamArgs(), "[K, V]", "args")
	})

	t.Run("StructInfo returns empty strings for non-generic", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "structs")
		s, err := pkg.Struct("Item")
		testkit.NoError(t, err, "Struct Item")
		tracker := generator.NewImportTracker(pkg.Path())
		testkit.Equal(t, s.TypeParamDecl(tracker), "", "non-generic decl")
		testkit.Equal(t, s.TypeParamArgs(), "", "non-generic args")
	})

	t.Run("InterfaceInfo returns empty strings for non-generic interface", func(t *testing.T) {
		t.Parallel()
		pkg := loadBasic(t)
		i, err := pkg.Interface("Store")
		testkit.NoError(t, err, "Interface Store")
		tracker := generator.NewImportTracker(pkg.Path())
		testkit.Equal(t, i.TypeParamDecl(tracker), "", "non-generic iface decl")
		testkit.Equal(t, i.TypeParamArgs(), "", "non-generic iface args")
	})
}

// loadFixture loads a fixture package by name from generator/testdata/.
// Mirrors the helper in builder/helpers_test.go for use across the
// generator's own external tests (loader/method/fields).
func loadFixture(t *testing.T, name string) *generator.Package {
	t.Helper()
	pkg, err := generator.NewLoader().Load("./testdata/"+name, "")
	testkit.NoError(t, err, "Load testdata/"+name)
	return pkg
}
