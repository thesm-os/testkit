// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package emitq_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sdk"
	"go.thesmos.sh/eidos/store"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/emitq"
)

// The three fields travel together, and a value missing any of them is one
// whose diagnostics point at the wrong source line — which is invisible until
// someone reads a failure and goes to the file it names.
func TestBase(t *testing.T) {
	t.Parallel()

	t.Run("carries the origin, the plugin and the position", func(t *testing.T) {
		t.Parallel()
		iface := ifaceFixture(t)
		base := emitq.Base(provenance(), iface)

		testkit.Equal(t, base.OriginNode, node.Node(iface), "the value knows what it came from")
		testkit.Equal(t, base.SetByName, "testplugin", "the value knows which plugin made it")
		testkit.Equal(t, base.SourcePos, iface.Pos(), "a diagnostic points at the declaration")
	})

	t.Run("leaves the output tag unset", func(t *testing.T) {
		t.Parallel()
		// An empty tag is the primary output. A base that arrived pre-tagged
		// would route a plugin's primary value to its companion's file.
		base := emitq.Base(provenance(), ifaceFixture(t))
		testkit.Equal(t, base.OutputTagName, "", "the primary output carries no tag")
	})
}

// Tagged returns a copy so a caller holding one base can derive several. A
// version that mutated would leave the primary pointing at the companion's
// output, which routes both files to one path.
func TestTagged(t *testing.T) {
	t.Parallel()

	t.Run("routes the copy to the named tag", func(t *testing.T) {
		t.Parallel()
		base := emitq.Base(provenance(), ifaceFixture(t))
		tagged := emitq.Tagged(base, "test")
		testkit.Equal(t, tagged.OutputTagName, "test", "the copy carries the tag")
	})

	t.Run("leaves the original untagged", func(t *testing.T) {
		t.Parallel()
		base := emitq.Base(provenance(), ifaceFixture(t))
		_ = emitq.Tagged(base, "test")
		testkit.Equal(t, base.OutputTagName, "", "the original is unchanged")
	})

	t.Run("keeps everything else", func(t *testing.T) {
		t.Parallel()
		base := emitq.Base(provenance(), ifaceFixture(t))
		tagged := emitq.Tagged(base, "test")
		testkit.Equal(t, tagged.OriginNode, base.OriginNode, "the origin survives")
		testkit.Equal(t, tagged.SetByName, base.SetByName, "the plugin survives")
		testkit.Equal(t, tagged.SourcePos, base.SourcePos, "the position survives")
	})
}

// The provenance id is what a later plugin targets to position its own
// contribution, and what `testkit explain` reports for "which plugin wrote
// this". Both break silently if it is composed differently in two places.
func TestAppend(t *testing.T) {
	t.Parallel()

	t.Run("queues every value against the origin's slot", func(t *testing.T) {
		t.Parallel()
		s, iface := storeAndIface(t)
		ctx := generatorContext(s)

		err := emitq.Append(ctx, provenance(), "top", iface,
			&fakeEmit{kind: "test.first"}, &fakeEmit{kind: "test.second"})
		testkit.NoError(t, err, "queueing two values must succeed")

		pending := s.Emit().PendingOriginSlots()
		testkit.Len(t, pending, 2, "both values are queued")
	})

	t.Run("names the provenance after the kind and the declaration", func(t *testing.T) {
		t.Parallel()
		s, iface := storeAndIface(t)

		err := emitq.Append(generatorContext(s), provenance(), "top", iface,
			&fakeEmit{kind: "test.first"})
		testkit.NoError(t, err, "queueing must succeed")

		pending := s.Emit().PendingOriginSlots()
		testkit.Len(t, pending, 1, "one value is queued")
		testkit.Equal(t, pending[0].Prov.ID, "test.first.Store",
			"the id names the kind and the declaration")
		testkit.Equal(t, pending[0].Prov.SetBy, "testplugin", "the id names the plugin")
	})

	t.Run("queues nothing when given nothing", func(t *testing.T) {
		t.Parallel()
		// A generator whose projection produced no values calls this with an
		// empty set rather than branching, so the empty case is the common one.
		s, iface := storeAndIface(t)
		err := emitq.Append(generatorContext(s), provenance(), "top", iface)
		testkit.NoError(t, err, "queueing nothing must succeed")
		testkit.Len(t, s.Emit().PendingOriginSlots(), 0, "nothing is queued")
	})
}

// AppendAs exists for package-scoped output, which hangs off whichever
// declaration the package happened to offer and is about the package instead.
func TestAppendAs(t *testing.T) {
	t.Parallel()

	t.Run("names the provenance after the caller's id", func(t *testing.T) {
		t.Parallel()
		// Deriving it from the anchor would move the identifier a plugin
		// targets when an unrelated type in the package is renamed.
		s, iface := storeAndIface(t)

		err := emitq.AppendAs(generatorContext(s), provenance(), "top", iface, "cfg",
			&fakeEmit{kind: "test.pkg"})
		testkit.NoError(t, err, "queueing must succeed")

		pending := s.Emit().PendingOriginSlots()
		testkit.Len(t, pending, 1, "one value is queued")
		testkit.Equal(t, pending[0].Prov.ID, "test.pkg.cfg",
			"the id names what the value is about, not what it hangs off")
	})
}

// Layout may hand back a partial map, and "no entry" and "empty entry" are the
// same answer to a caller: there is no package to qualify against. Folding them
// here is what stops each implementation reasoning about it again.
func TestPrimaryPackage(t *testing.T) {
	t.Parallel()

	t.Run("returns the primary output's package", func(t *testing.T) {
		t.Parallel()
		path, ok := emitq.PrimaryPackage(map[string]string{"": "example.com/x"})
		testkit.True(t, ok, "a resolved primary is reported")
		testkit.Equal(t, path, "example.com/x", "the package is returned")
	})

	t.Run("declines a map carrying no primary", func(t *testing.T) {
		t.Parallel()
		// A run that recorded routing errors reaches dispatch with tags missing.
		_, ok := emitq.PrimaryPackage(map[string]string{"test": "example.com/x"})
		testkit.False(t, ok, "a map without the primary tag resolves nothing")
	})

	t.Run("declines a primary that resolved to nothing", func(t *testing.T) {
		t.Parallel()
		// Present but empty is the shape layout produces for a target it could
		// not compose, and treating it as a package qualifies against "".
		_, ok := emitq.PrimaryPackage(map[string]string{"": ""})
		testkit.False(t, ok, "an empty primary resolves nothing")
	})

	t.Run("declines an empty map", func(t *testing.T) {
		t.Parallel()
		_, ok := emitq.PrimaryPackage(map[string]string{})
		testkit.False(t, ok, "an empty map resolves nothing")
	})
}

// fakeEmit is the smallest thing the store accepts, so these tests exercise
// queueing rather than any generator's projection.
type fakeEmit struct {
	sdk.BaseEmit
	kind sdk.Kind
}

func (f *fakeEmit) Kind() sdk.Kind { return f.kind }

func provenance() *sdk.Provenance {
	return sdk.NewProvenance("testplugin", sdk.EmitTarget{})
}

func storeAndIface(t *testing.T) (*store.Store, *node.Interface) {
	t.Helper()
	iface := ifaceFixture(t)
	s := store.New()
	pkg := &node.Package{Name: "cfg", Path: "example.com/cfg", Interfaces: []*node.Interface{iface}}
	if err := s.Nodes().AddPackage(pkg); err != nil {
		t.Fatalf("seed the store: %v", err)
	}
	return s, iface
}

func ifaceFixture(t *testing.T) *node.Interface {
	t.Helper()
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Interface("Store", func(b *storefixture.InterfaceBuilder) {
			b.Method("Get", func(*storefixture.MethodBuilder) {})
		}).
		PackageNode().Interfaces[0]
}

func generatorContext(s *store.Store) *plugin.GeneratorContext {
	return &plugin.GeneratorContext{Store: s, Reader: store.NewReader(s), Diag: diag.New()}
}
