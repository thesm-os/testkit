// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub_test

import (
	"io/fs"
	"testing"

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
		_, ok := stub.GoTemplates()
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

// The shared helpers are what the templates call, and an empty funcmap
// surfaces as a template execution error rather than as missing output.
func TestGoFuncMap(t *testing.T) {
	t.Parallel()

	t.Run("contributes the shared list helpers", func(t *testing.T) {
		t.Parallel()
		testkit.Assert(t, stub.GoFuncMap()).IsNotEmpty("the adapter contributes the shared Go helpers")
	})

	t.Run("registers every entry under the plugin's own prefix", func(t *testing.T) {
		t.Parallel()
		// The backend fails the whole run when two plugins register one
		// extension name, so an unprefixed entry here breaks every output the
		// moment a second testkit generator contributes the same bundle.
		for name := range stub.GoFuncMap() {
			testkit.HasPrefix(t, name, stub.Name, "every funcmap entry is namespaced")
		}
	})
}

// hasTemplate reports whether name resolves against the adapter's embedded
// tree. A missing template renders nothing and fails nowhere, so presence is
// the whole assertion — the stat error itself carries nothing a reader needs.
func hasTemplate(t *testing.T, name string) bool {
	t.Helper()
	tree, ok := stub.GoTemplates()
	if !ok {
		t.Fatal("GoTemplates reported no tree")
	}
	_, err := fs.Stat(tree, name)
	return err == nil
}
