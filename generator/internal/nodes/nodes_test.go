// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package nodes_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/node"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/nodes"
)

// Reading the name off the reference rather than off the pointee is what made
// an embed by pointer vanish from a generated builder without a diagnostic, so
// both spellings are pinned here rather than only the one that was broken.
func TestEmbedName(t *testing.T) {
	t.Parallel()

	t.Run("takes the name of a type embedded by value", func(t *testing.T) {
		t.Parallel()
		name, pointer := nodes.EmbedName(storefixture.Named("Meta"))
		testkit.Equal(t, name, "Meta", "the reference names itself")
		testkit.False(t, pointer, "an embed by value is not a pointer")
	})

	t.Run("takes the name of a type embedded by pointer from its pointee", func(t *testing.T) {
		t.Parallel()
		// The reference carries no name of its own, so reading it yields the
		// empty string and the caller drops the field.
		name, pointer := nodes.EmbedName(storefixture.Pointer(storefixture.Named("Audit")))
		testkit.Equal(t, name, "Audit", "the pointee names the field")
		testkit.True(t, pointer, "an embed by pointer is reported as one")
	})

	t.Run("declines an embed with no recorded type", func(t *testing.T) {
		t.Parallel()
		name, _ := nodes.EmbedName(nil)
		testkit.Equal(t, name, "", "an embed with no type contributes no field")
	})

	t.Run("declines a pointer with no recorded pointee", func(t *testing.T) {
		t.Parallel()
		// Composing a setter from a name that is not there would emit a method
		// on a type the file never declares.
		name, _ := nodes.EmbedName(&node.TypeRef{TypeKind: node.TypeRefPointer})
		testkit.Equal(t, name, "", "a pointer to nothing contributes no field")
	})
}

// The node model has no channel kind, so every channel arrives as the same
// named reference and only the stamp tells them apart.
func TestIsBidirectionalChan(t *testing.T) {
	t.Parallel()

	t.Run("accepts a bidirectional channel", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, nodes.IsBidirectionalChan(chanRef(nodes.ChanBidirectional)),
			"a channel that can be sent to and received from is accepted")
	})

	t.Run("declines a directional channel", func(t *testing.T) {
		t.Parallel()
		// make is not legal on one, and a caller asks this in order to write a
		// make.
		testkit.False(t, nodes.IsBidirectionalChan(chanRef("recv")),
			"a receive-only channel is declined")
	})

	t.Run("declines a direction it does not recognise", func(t *testing.T) {
		t.Parallel()
		// Matched positively: an unknown spelling is likelier to be one make
		// rejects than one it accepts.
		testkit.False(t, nodes.IsBidirectionalChan(chanRef("sideways")),
			"an unrecognised direction is declined")
	})

	t.Run("declines a reference carrying no stamp", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, nodes.IsBidirectionalChan(storefixture.Named("string")),
			"an ordinary type is not a channel")
	})

	t.Run("declines an absent reference", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, nodes.IsBidirectionalChan(nil), "nothing is not a channel")
	})
}

// Each of these is two lines, which is exactly why a caller that cannot find
// them writes a private copy that gets an edge case wrong.
func TestTypePredicates(t *testing.T) {
	t.Parallel()

	t.Run("recognises the builtin error", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, nodes.IsError(storefixture.Named("error")), "error is the builtin")
		testkit.False(t, nodes.IsError(storefixture.PkgNamed("x", "error")), "a foreign error is not it")
		testkit.False(t, nodes.IsError(nil), "nothing is not an error")
	})

	t.Run("recognises both spellings of byte", func(t *testing.T) {
		t.Parallel()
		// The edge case a private copy misses: the frontend records whichever
		// the author wrote.
		testkit.True(t, nodes.IsByte(storefixture.Named("byte")), "byte is byte")
		testkit.True(t, nodes.IsByte(storefixture.Named("uint8")), "uint8 is byte")
		testkit.False(t, nodes.IsByte(storefixture.Named("int8")), "int8 is not byte")
	})

	t.Run("recognises the anonymous empty struct", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, nodes.IsEmptyStruct(storefixture.AnonStruct(nil, nil)), "struct{} is empty")
		testkit.False(t, nodes.IsEmptyStruct(storefixture.Named("Unit")), "a named type is not it")
	})

	t.Run("declines an anonymous struct carrying a field", func(t *testing.T) {
		t.Parallel()
		// A struct holding anything is a value a caller has something to say
		// about, so it is not the set marker.
		held := storefixture.AnonStruct([]*node.Field{{Name: "N"}}, nil)
		testkit.False(t, nodes.IsEmptyStruct(held), "a populated struct is not empty")
	})

	t.Run("recognises the empty interface", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, nodes.IsEmptyInterface(storefixture.AnonInterface(nil, nil)), "interface{} is any")
		testkit.False(t, nodes.IsEmptyInterface(storefixture.AnonInterface([]*node.Method{{Name: "Read"}}, nil)),
			"an interface with a method is not any")
	})
}

// A method's presence and its receiver form are separate questions, and the
// second one silently decides whether errors.Is ever reaches a declared Is.
func TestMethodQueries(t *testing.T) {
	t.Parallel()

	t.Run("reports a declared method", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, nodes.Declares(errType(t), "Error"), "Error is declared")
		testkit.False(t, nodes.Declares(errType(t), "Unwrap"), "Unwrap is not")
	})

	t.Run("reports the receiver form", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, nodes.PointerReceiver(errType(t), "Error"), "Error is on the pointer")
		testkit.False(t, nodes.PointerReceiver(errType(t), "Is"), "Is is on the value")
	})

	t.Run("reports the value form for a method that is absent", func(t *testing.T) {
		t.Parallel()
		// A caller that has not checked Declares gets the value form rather
		// than a claim about a method that is not there.
		testkit.False(t, nodes.PointerReceiver(errType(t), "Unwrap"), "an absent method is not a pointer")
	})

	t.Run("tolerates an absent type", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, nodes.Declares(nil, "Error"), "nothing declares nothing")
		testkit.False(t, nodes.PointerReceiver(nil, "Error"), "nothing has no receiver")
		testkit.Equal(t, nodes.FieldOfType(nil, "error"), "", "nothing has no fields")
	})

	t.Run("finds the first exported field of a type", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, nodes.FieldOfType(errType(t), "error"), "Cause", "the error field is found")
		testkit.Equal(t, nodes.FieldOfType(errType(t), "float64"), "", "an absent type yields nothing")
	})

	t.Run("skips an unexported field", func(t *testing.T) {
		t.Parallel()
		// A caller cannot name it from another package, so reporting it would
		// produce a literal that does not compile.
		testkit.Equal(t, nodes.FieldOfType(errType(t), "string"), "Op",
			"the unexported string is skipped for the exported one")
	})
}

// chanRef builds a channel the way the Go frontend does: a named reference in a
// synthetic `go` package, with the facts that make it a channel stamped beside
// it rather than expressed in its shape.
func chanRef(dir string) *node.TypeRef {
	ref := storefixture.WithArgs(storefixture.PkgNamed("go", "chan"), storefixture.Named("string"))
	nodes.GoIsChannel.Set(ref.EnsureMeta(), true, "golang")
	nodes.GoChanDir.Set(ref.EnsureMeta(), dir, "golang")
	return ref
}

// errType returns a struct shaped like a custom error: Error on the pointer,
// Is on the value, an unexported field ahead of the exported ones.
func errType(t *testing.T) *node.Struct {
	t.Helper()
	pkg := storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("WrappedError", func(b *storefixture.StructBuilder) {
			b.Field("hidden", storefixture.Named("string"), nil)
			b.Field("Op", storefixture.Named("string"), nil)
			b.Field("Cause", storefixture.Named("error"), nil)
		}).
		PackageNode()
	s := pkg.Structs[0]
	s.Methods = []*node.Method{
		{Name: "Error", Receiver: storefixture.Pointer(storefixture.Named("WrappedError"))},
		{Name: "Is", Receiver: storefixture.Named("WrappedError")},
	}
	_ = meta.AuthorityPlugin
	return s
}
