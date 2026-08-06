// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gotmpl_test

import (
	"testing"

	"go.thesmos.sh/eidos/emit"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/gotmpl"
	"go.thesmos.sh/testkit/generator/internal/signature"
)

// Ref is what keeps a module path out of every template, so the mapping from
// alias to path is the assertion — and a symbol nobody can resolve has to
// fail rather than resolve to something plausible.
func TestRef(t *testing.T) {
	t.Parallel()

	t.Run("resolves the runtime module", func(t *testing.T) {
		t.Parallel()
		got, err := gotmpl.Ref("testkit.ErrorIs")
		testkit.NoError(t, err, "a known alias resolves")
		assertExternal(t, got, gotmpl.Module, "ErrorIs")
	})

	t.Run("resolves a runtime subpackage", func(t *testing.T) {
		t.Parallel()
		got, err := gotmpl.Ref("stub.Behaviour")
		testkit.NoError(t, err, "a known alias resolves")
		assertExternal(t, got, gotmpl.Module+"/stub", "Behaviour")
	})

	t.Run("resolves a standard library package", func(t *testing.T) {
		t.Parallel()
		// The stdlib aliases earn their place by keeping the templates on one
		// spelling: a template that wrote `external "testing" "T"` for some
		// symbols and used the table for the rest would drift.
		got, err := gotmpl.Ref("testing.TB")
		testkit.NoError(t, err, "a known alias resolves")
		assertExternal(t, got, "testing", "TB")
	})

	t.Run("rejects an unknown package", func(t *testing.T) {
		t.Parallel()
		// A silent fallback here would emit a reference qualified by a package
		// the rendered file never imports, which fails at compile time in the
		// consumer's repository rather than in ours.
		_, err := gotmpl.Ref("nowhere.Thing")
		testkit.ErrorIs(t, err, gotmpl.ErrBadSymbol, "an unknown alias must fail the render")
	})

	t.Run("rejects a symbol with no package", func(t *testing.T) {
		t.Parallel()
		_, err := gotmpl.Ref("ErrorIs")
		testkit.ErrorIs(t, err, gotmpl.ErrBadSymbol, "a bare symbol must fail the render")
	})

	t.Run("rejects a package with no symbol", func(t *testing.T) {
		t.Parallel()
		_, err := gotmpl.Ref("testkit.")
		testkit.ErrorIs(t, err, gotmpl.ErrBadSymbol, "an empty symbol must fail the render")
	})

	t.Run("names the alternatives it knows", func(t *testing.T) {
		t.Parallel()
		// The message is the whole remedy for a typo, so it carries the set
		// rather than only reporting that the alias was wrong.
		_, err := gotmpl.Ref("nowhere.Thing")
		testkit.Contains(t, err.Error(), "testkit", "the diagnostic lists the known aliases")
	})
}

// The list renderers are what every generated signature is spelled from, so
// each is checked at the arities a real method reaches: none, one, several.
func TestLists(t *testing.T) {
	t.Parallel()

	t.Run("renders an empty list as nothing", func(t *testing.T) {
		t.Parallel()
		// An empty list is the void method's argument list and the no-return
		// method's tuple. Rendering a stray separator there would not compile.
		testkit.Equal(t, gotmpl.Args(nil), "", "no parameters render nothing")
		testkit.Equal(t, gotmpl.Locals(nil), "", "no returns render nothing")
		testkit.Equal(t, gotmpl.Idents("a", 0), "", "no identifiers render nothing")
	})

	t.Run("renders a single entry without a separator", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gotmpl.Args(params()[:1]), "ctx", "one parameter renders bare")
	})

	t.Run("separates several entries", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gotmpl.Args(params()), "ctx, id", "parameters render as an argument list")
	})

	t.Run("assigns parameters to their recorded-call fields", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gotmpl.CallFields(params()), "Ctx: ctx, ID: id",
			"a recorded call is built field by field")
	})

	t.Run("binds returns to their capture locals", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gotmpl.Locals(returns()), "r0, r1", "captures bind positionally")
	})

	t.Run("assigns captures to their return fields", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gotmpl.LocalFields(returns()), "Result: r0, Err: r1",
			"a return tuple is built from the captures")
	})

	t.Run("assigns positional identifiers to their return fields", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gotmpl.IdentFields("got", returns()), "Result: got0, Err: got1",
			"a checked answer is boxed from the call's results")
	})

	t.Run("names the consumer-facing setter parameters", func(t *testing.T) {
		t.Parallel()
		// Returns is public surface, so its parameters read as the fields they
		// set rather than as the internal capture locals.
		testkit.Equal(t, gotmpl.NamedFields(returns()), "Result: result, Err: err",
			"the setter assigns from its own parameter names")
	})

	t.Run("reads a resolved answer back off its tuple", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gotmpl.Reads(returns()), "r.Result, r.Err",
			"the dispatch body returns the answer's fields")
	})

	t.Run("numbers positional identifiers from zero", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gotmpl.Idents("want", 3), "want0, want1, want2",
			"identifiers are numbered from zero")
	})

	t.Run("renders one discard per slot", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gotmpl.Blanks(2), "_, _", "every slot is discarded")
	})
}

// Forwarding a variadic parameter without its ellipsis passes the slice as a
// single element, which type-checks and silently records one argument where
// the caller passed several.
func TestVariadic(t *testing.T) {
	t.Parallel()

	t.Run("spreads a variadic tail in an argument list", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gotmpl.Args(variadic()), "ctx, keys...", "the tail is spread")
	})

	t.Run("spreads a variadic tail in a positional argument list", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gotmpl.IdentArgs("a", variadic()), "a0, a1...", "the tail is spread")
	})

	t.Run("leaves a fixed list alone", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gotmpl.IdentArgs("a", params()), "a0, a1", "nothing variadic, nothing spread")
	})

	t.Run("renders an empty list as nothing", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gotmpl.IdentArgs("a", nil), "", "no parameters render nothing")
	})
}

// Fails is the one list that reads the projection rather than only reshaping
// it, and reading it wrong produces a check that compiles and asserts nothing.
func TestFails(t *testing.T) {
	t.Parallel()

	t.Run("binds the error slot and discards the rest", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gotmpl.Fails(returns()), "_, r1", "only the error is bound")
	})

	t.Run("finds the error slot wherever it sits", func(t *testing.T) {
		t.Parallel()
		// `(error, string)` is unusual but legal Go, and a rule that assumed
		// the last slot would bind the value and assert on nothing.
		leading := []signature.Return{
			{Field: "Err", Local: "r0", Error: true},
			{Field: "Result", Local: "r1"},
		}
		testkit.Equal(t, gotmpl.Fails(leading), "r0, _", "the error is found by flag, not by position")
	})

	t.Run("discards everything when nothing can fail", func(t *testing.T) {
		t.Parallel()
		values := []signature.Return{{Field: "Result", Local: "r0"}}
		testkit.Equal(t, gotmpl.Fails(values), "_", "a signature with no error binds nothing")
	})
}

// A helper missing from the funcmap surfaces as a template execution error
// rather than as missing output, and the prefix is what keeps two plugins
// from colliding at Build time.
func TestFuncMap(t *testing.T) {
	t.Parallel()

	t.Run("registers every helper under the prefix", func(t *testing.T) {
		t.Parallel()
		fm := gotmpl.FuncMap("stub")
		for _, name := range []string{
			"stubArgs", "stubBlanks", "stubCallFields", "stubFails", "stubIdentFields",
			"stubIdents", "stubLocalFields", "stubLocals", "stubNamedFields", "stubReads",
			"stubRef",
		} {
			_, ok := fm[name]
			testkit.True(t, ok, "the funcmap carries "+name)
		}
	})

	t.Run("shares no name between two prefixes", func(t *testing.T) {
		t.Parallel()
		// The backend rejects two plugins registering the same extension
		// outright, so an overlap here fails every run rather than one output.
		other := gotmpl.FuncMap("fault")
		for name := range gotmpl.FuncMap("stub") {
			_, clash := other[name]
			testkit.False(t, clash, "prefixes must not overlap on "+name)
		}
	})
}

// assertExternal checks that ref is an external reference to name in path.
//
// The kind is asserted first: every other expression variant leaves Pkg empty,
// so a wrong kind would otherwise fail on a path comparison that says nothing
// about what actually went wrong.
func assertExternal(t *testing.T, ref *emit.Expr, path, name string) {
	t.Helper()
	testkit.Equal(t, ref.ExprKind, emit.ExprExternal, "a resolved symbol is a package-qualified reference")
	testkit.Equal(t, ref.Pkg, path, "the alias resolves to its import path")
	testkit.Equal(t, ref.Name, name, "the symbol half is carried through")
}

// params is the projection of `(ctx context.Context, id string)`, which is the
// shape most methods in the corpus take.
func params() []signature.Param {
	return []signature.Param{
		{Name: "ctx", Field: "Ctx"},
		{Name: "id", Field: "ID"},
	}
}

// variadic is the projection of `(ctx context.Context, keys ...string)`.
func variadic() []signature.Param {
	return []signature.Param{
		{Name: "ctx", Field: "Ctx"},
		{Name: "keys", Field: "Keys", Variadic: true},
	}
}

// returns is the projection of `(string, error)` — one value slot and one
// error slot, which is what every fault-bearing method has.
func returns() []signature.Return {
	return []signature.Return{
		{Field: "Result", Local: "r0"},
		{Field: "Err", Local: "r1", Error: true},
	}
}
