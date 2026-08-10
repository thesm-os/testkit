// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub_test

import (
	"io/fs"
	"maps"
	"slices"
	"testing"

	"go.thesmos.sh/eidos/lang/golang"
	sdkgolang "go.thesmos.sh/eidos/sdk/golang"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/stub"
)

// The Go adapter's output set is what Layout appends to each source
// basename, so the suffixes are a naming contract with every consumer that
// has ever run the generator.
func TestGoOutputs(t *testing.T) {
	t.Parallel()

	t.Run("declares both the primary and the companion", func(t *testing.T) {
		t.Parallel()
		testkit.Len(t, stub.GoOutputs(), 2, "the adapter declares a pair")
	})

	t.Run("marks generated files so tooling can skip them", func(t *testing.T) {
		t.Parallel()
		testkit.HasSuffix(t, stub.GoPrimarySuffix, ".gen.go", "the primary is marked generated")
	})

	t.Run("ends the companion in _test.go so the package shift fires", func(t *testing.T) {
		t.Parallel()
		// The framework keys the external-test-package shift off this exact
		// ending. A suffix that merely looked test-ish would land the suite
		// in the source package, where it could read private state and would
		// no longer prove the double works from outside.
		testkit.HasSuffix(t, stub.GoTestSuffix, "_test.go", "the companion triggers the shift")
	})
}

// A template the backend cannot find renders nothing and fails nowhere, so
// both entry points are checked by name rather than assumed present.
func TestGoTemplates(t *testing.T) {
	t.Parallel()

	t.Run("reports a tree", func(t *testing.T) {
		t.Parallel()
		_, ok := stub.New().Templates(golang.Language)
		testkit.True(t, ok, "the embedded tree is always available")
	})

	t.Run("carries the template the double renders from", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, hasTemplate(t, "stub.double.tmpl"), "the primary template must ship")
	})

	t.Run("carries the template the companion renders from", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, hasTemplate(t, "stub.test.tmpl"), "the companion template must ship")
	})
}

// hasTemplate reports whether the named template ships in the plugin's Go
// tree.
//
// Looked up by base name because that is how the backend registers them: a
// tree rooted one directory too high still resolves the file by a longer path
// and still renders nothing.
func hasTemplate(t *testing.T, name string) bool {
	t.Helper()
	tree, ok := stub.New().Templates(golang.Language)
	if !ok {
		t.Fatal("the plugin reports no Go template tree")
	}
	_, err := fs.Stat(tree, name)
	return err == nil
}

// The shared helpers are what the templates call, and an empty funcmap
// surfaces as a template execution error rather than as missing output.
func TestGoFuncMap(t *testing.T) {
	t.Parallel()

	t.Run("contributes the shared list helpers", func(t *testing.T) {
		t.Parallel()
		funcs := stub.New().TemplateFuncs(golang.Language)
		testkit.Assert(t, funcs).IsNotEmpty("the adapter contributes the shared Go helpers")
	})

	t.Run("registers every entry under the plugin's own prefix", func(t *testing.T) {
		t.Parallel()
		// The backend fails the whole run when two plugins register one
		// extension name, so an unprefixed entry here breaks every output the
		// moment a second testkit generator contributes the same bundle.
		for name := range stub.New().TemplateFuncs(golang.Language) {
			testkit.HasPrefix(t, name, stub.Name, "every funcmap entry is namespaced")
		}
	})

	t.Run("adds no helper of its own to the shared bundle", func(t *testing.T) {
		t.Parallel()
		// A `ref` helper used to live here, resolving testkit's import paths
		// against a private table. The backend's own `external` does exactly
		// that, so what the plugin registers is now the shared bundle and
		// nothing else. Pinned because a helper creeping back in is a piece of
		// Go knowledge moving out of eidos and into this repository, which is
		// invisible in any assertion about what the templates produce.
		shared := golang.AllFuncMap(sdkgolang.FuncPrefix(stub.Name))
		got := stub.New().TemplateFuncs(golang.Language)
		testkit.Len(t, got, len(shared), "the plugin contributes exactly the shared bundle")
		for name := range got {
			testkit.Contains(t, slices.Sorted(maps.Keys(shared)), name,
				"every registered helper comes from the shared bundle")
		}
	})
}
