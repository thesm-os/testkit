// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package signature_test

import (
	"testing"

	"go.thesmos.sh/eidos/node"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/signature"
)

func TestParamsOf(t *testing.T) {
	t.Parallel()

	t.Run("keeps the declared identifier", func(t *testing.T) {
		t.Parallel()
		got := signature.ParamsOf(method(params(param("ctx", "context", "Context")), nil))
		testkit.Equal(t, got[0].Name, "ctx", "a named parameter keeps its name")
	})

	t.Run("names an unnamed parameter positionally", func(t *testing.T) {
		t.Parallel()
		// The generated body has to reference the parameter when recording
		// the call, so every slot needs an identifier whether or not the
		// source supplied one.
		got := signature.ParamsOf(method(params(param("", "", "int")), nil))
		testkit.Equal(t, got[0].Name, "arg0", "an unnamed parameter takes a positional name")
	})

	t.Run("exports the recorded-call field name", func(t *testing.T) {
		t.Parallel()
		got := signature.ParamsOf(method(params(param("ctx", "context", "Context")), nil))
		testkit.Equal(t, got[0].Field, "Ctx", "the field is the exported form")
	})

	t.Run("expands an initialism in the field name", func(t *testing.T) {
		t.Parallel()
		// `Id` reads as a typo to a Go reader, and the recorded call is what
		// a failure message prints.
		got := signature.ParamsOf(method(params(param("id", "", "string")), nil))
		testkit.Equal(t, got[0].Field, "ID", "id exports as ID")
	})
}

func TestReturnsOf(t *testing.T) {
	t.Parallel()

	t.Run("derives the field name from a declared return name", func(t *testing.T) {
		t.Parallel()
		// A signature written `(item User, err error)` documents what its
		// returns mean, and the recorded-call struct is the main consumer of
		// that documentation.
		got := signature.ReturnsOf(method(nil, returns(ret("item", "", "User"))))
		testkit.Equal(t, got[0].Field, "Item", "a named return names its field")
	})

	t.Run("names an unnamed error slot Err", func(t *testing.T) {
		t.Parallel()
		// Every generated assertion names this field, and `Result1` would
		// say nothing about what it holds.
		got := signature.ReturnsOf(method(nil, returns(ret("", "", "error"))))
		testkit.Equal(t, got[0].Field, "Err", "the error slot is named for what it is")
	})

	t.Run("names a lone unnamed value slot Result", func(t *testing.T) {
		t.Parallel()
		// An index distinguishes it from nothing when it is the only value.
		got := signature.ReturnsOf(method(nil, returns(ret("", "", "User"), ret("", "", "error"))))
		testkit.Equal(t, got[0].Field, "Result", "a single value slot needs no index")
	})

	t.Run("indexes several unnamed value slots", func(t *testing.T) {
		t.Parallel()
		got := signature.ReturnsOf(method(nil, returns(ret("", "", "User"), ret("", "", "bool"))))
		testkit.Equal(t, got[1].Field, "Result1", "several value slots are indexed")
	})

	t.Run("numbers value slots independently of the error slot", func(t *testing.T) {
		t.Parallel()
		// Adding an error return must not renumber the fields beside it, or
		// every consumer reading a recorded call breaks on an unrelated
		// signature change.
		got := signature.ReturnsOf(method(nil, returns(ret("", "", "User"), ret("", "", "error"), ret("", "", "bool"))))
		testkit.Equal(t, got[2].Field, "Result1", "the error slot does not consume an index")
	})

	t.Run("indexes a second error slot rather than duplicating Err", func(t *testing.T) {
		t.Parallel()
		// Two errors is legal and vanishingly rare, but a duplicate field
		// name would not compile.
		got := signature.ReturnsOf(method(nil, returns(ret("", "", "error"), ret("", "", "error"))))
		testkit.Equal(t, got[1].Field, "Err1", "a second error slot is indexed")
	})

	t.Run("gives every slot a field even when only some are named", func(t *testing.T) {
		t.Parallel()
		// Field naming is per-return and has no all-or-nothing constraint,
		// unlike the signature's own return names.
		got := signature.ReturnsOf(method(nil, returns(ret("item", "", "User"), ret("", "", "error"))))
		testkit.Equal(t, got[1].Field, "Err", "the unnamed slot still gets a field")
	})

	t.Run("carries the declared name through unchanged", func(t *testing.T) {
		t.Parallel()
		got := signature.ReturnsOf(method(nil, returns(ret("item", "", "User"))))
		testkit.Equal(t, got[0].Name, "item", "the declared name is preserved for the signature")
	})
}

// The named-return decision is all-or-nothing, and every way of failing it
// would otherwise produce a signature that does not compile. Table-driven
// because these are independent reasons for one answer over uniform inputs.
func TestNamedReturnsUsable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		method *node.Method
		want   bool
	}{
		{
			name:   "propagates when every return is named",
			method: method(params(param("id", "", "string")), returns(ret("item", "", "User"), ret("err", "", "error"))),
			want:   true,
		},
		{
			name:   "declines when there are no returns",
			method: method(nil, nil),
			want:   false,
		},
		{
			// `(item User, _ error)` is valid Go, and the blank normalises to
			// unnamed — so the model holds one named and one unnamed slot,
			// which the emit layer rejects outright.
			name:   "declines when one return is left unnamed",
			method: method(nil, returns(ret("item", "", "User"), ret("", "", "error"))),
			want:   false,
		},
		{
			// `func (s *T) F() (s int)` does not compile.
			name:   "declines when a return collides with the receiver",
			method: method(nil, returns(ret(signature.ReceiverIdent, "", "int"))),
			want:   false,
		},
		{
			// `func (s *T) F(item int) (item int)` does not compile either.
			name:   "declines when a return collides with a parameter",
			method: method(params(param("item", "", "int")), returns(ret("item", "", "int"))),
			want:   false,
		},
		{
			// The positional fallback for an unnamed parameter is a real
			// identifier and collides just the same.
			name:   "declines when a return collides with a positional parameter",
			method: method(params(param("", "", "int")), returns(ret("arg0", "", "int"))),
			want:   false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testkit.Equal(t, signature.NamedReturnsUsable(tc.method), tc.want, "named-return decision")
		})
	}
}

func TestWithLocals(t *testing.T) {
	t.Parallel()

	t.Run("reuses the declared name for a named result", func(t *testing.T) {
		t.Parallel()
		// Named results are already declared by the signature, so the local
		// is the name and the body assigns rather than declares.
		m := method(nil, returns(ret("item", "", "User")))
		got := signature.WithLocals(signature.ReturnsOf(m), nil, true)
		testkit.Equal(t, got[0].Local, "item", "a named result binds to its own name")
	})

	t.Run("takes a positional local for an anonymous result", func(t *testing.T) {
		t.Parallel()
		m := method(nil, returns(ret("", "", "User"), ret("", "", "error")))
		got := signature.WithLocals(signature.ReturnsOf(m), nil, false)
		testkit.Equal(t, got[1].Local, "r1", "anonymous results bind positionally")
	})

	t.Run("prefixes a local that would shadow a parameter", func(t *testing.T) {
		t.Parallel()
		// Shadowing the parameter would record the wrong value, which is a
		// silent wrong answer rather than a compile error.
		m := method(params(param("r0", "", "int")), returns(ret("", "", "error")))
		got := signature.WithLocals(signature.ReturnsOf(m), signature.ParamsOf(m), false)
		testkit.Equal(t, got[0].Local, "_r0", "the local must not shadow the parameter")
	})
}

// A range-over-func return is what earns a method its Yields helpers, and
// misclassifying one emits a helper that does not compile.
func TestIteratorOf(t *testing.T) {
	t.Parallel()

	t.Run("classifies a single-argument sequence", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, signature.IteratorOf(seq("Seq", named("", "Value"))),
			signature.SeqIterator, "iter.Seq[V] is a sequence")
	})

	t.Run("classifies a two-argument sequence", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, signature.IteratorOf(seq("Seq2", named("", "Value"), named("", "error"))),
			signature.Seq2Iterator, "iter.Seq2[V, error] is a sequence")
	})

	t.Run("declines a type from another package", func(t *testing.T) {
		t.Parallel()
		// A consumer's own two-parameter generic is not a sequence, and
		// treating it as one would emit a yield closure for a type that has
		// no such shape.
		other := &node.TypeRef{Package: "example.com/x", Name: "Seq2", TypeArgs: []*node.TypeRef{
			named("", "Value"), named("", "error"),
		}}
		testkit.Equal(t, signature.IteratorOf(other), signature.NotIterator, "package must be iter")
	})

	t.Run("declines a sequence with the wrong argument count", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, signature.IteratorOf(seq("Seq", named("", "K"), named("", "V"))),
			signature.NotIterator, "iter.Seq takes exactly one argument")
	})

	t.Run("declines a nil reference", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, signature.IteratorOf(nil), signature.NotIterator, "a void method has no sequence")
	})
}

func TestIteratorAccessors(t *testing.T) {
	t.Parallel()

	errSeq := seq("Seq2", named("", "Value"), named("", "error"))
	pairSeq := seq("Seq2", named("", "Key"), named("", "Value"))

	t.Run("reports a sequence that can fail partway through", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, signature.IteratorYieldsError(errSeq), "the second argument is error")
	})

	t.Run("declines a pair sequence that cannot fail", func(t *testing.T) {
		t.Parallel()
		// A key-value sequence has no error slot to append a failure to, so a
		// YieldsError helper would have nowhere to put one.
		testkit.False(t, signature.IteratorYieldsError(pairSeq), "a pair sequence carries no error")
	})

	t.Run("takes the element type from the first argument", func(t *testing.T) {
		t.Parallel()
		testkit.Assert(t, signature.IteratorElem(errSeq)).IsNotNil("the element is what a caller collects")
	})

	t.Run("reports no element for a non-sequence", func(t *testing.T) {
		t.Parallel()
		testkit.Assert(t, signature.IteratorElem(named("", "string"))).IsNil("a plain type has no element")
	})

	t.Run("takes the second argument from a pair sequence", func(t *testing.T) {
		t.Parallel()
		testkit.Assert(t, signature.IteratorSecond(errSeq)).IsNotNil("Seq2 has a second argument")
	})

	t.Run("reports no second argument for a single-argument sequence", func(t *testing.T) {
		t.Parallel()
		testkit.Assert(t, signature.IteratorSecond(seq("Seq", named("", "Value")))).
			IsNil("iter.Seq has only one argument")
	})
}

// seq builds a reference to a stdlib range-over-func sequence.
func seq(name string, args ...*node.TypeRef) *node.TypeRef {
	return &node.TypeRef{Package: "iter", Name: name, TypeArgs: args}
}

// named builds a plain type reference.
func named(pkg, name string) *node.TypeRef {
	return &node.TypeRef{Package: pkg, Name: name}
}

// method assembles a source method from parameter and return slots.
func method(ps []*node.Param, rs []*node.Return) *node.Method {
	return &node.Method{Name: "Do", Params: ps, Returns: rs}
}

func params(p ...*node.Param) []*node.Param { return p }

func returns(r ...*node.Return) []*node.Return { return r }

func param(name, pkg, typ string) *node.Param {
	return &node.Param{Name: name, Type: &node.TypeRef{Package: pkg, Name: typ}}
}

func ret(name, pkg, typ string) *node.Return {
	return &node.Return{Name: name, Type: &node.TypeRef{Package: pkg, Name: typ}}
}
