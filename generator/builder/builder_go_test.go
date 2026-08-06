// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"io/fs"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/builder"
)

// The Go adapter's output set is what Layout appends to each source basename,
// so the suffixes are a naming contract with every consumer that has run the
// generator.
func TestGoOutputs(t *testing.T) {
	t.Parallel()

	t.Run("declares the builder and its checks", func(t *testing.T) {
		t.Parallel()
		testkit.Len(t, builder.GoOutputs(), 2, "the adapter declares a pair")
	})

	t.Run("marks generated files so tooling can skip them", func(t *testing.T) {
		t.Parallel()
		testkit.HasSuffix(t, builder.GoPrimarySuffix, ".gen.go", "the primary is marked generated")
	})

	t.Run("ends the companion in _test.go so the package shift fires", func(t *testing.T) {
		t.Parallel()
		// The framework keys the external-test-package shift off this exact
		// ending. A suffix that merely looked test-ish would land the checks in
		// the builder's own package, where they would no longer drive it the
		// way a consumer does.
		testkit.HasSuffix(t, builder.GoTestSuffix, "_test.go", "the companion triggers the shift")
	})
}

// A template the backend cannot find renders nothing and fails nowhere, so both
// entry points are checked by name rather than assumed present.
func TestGoTemplates(t *testing.T) {
	t.Parallel()

	t.Run("carries the template the builder renders from", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, hasTemplate(t, "builder.type.tmpl"), "the primary template must ship")
	})

	t.Run("carries the template the checks render from", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, hasTemplate(t, "builder.test.tmpl"), "the companion template must ship")
	})
}

// The funcmap is where this plugin and its neighbours must not collide: the
// backend fails the whole run when two register one extension name.
func TestGoFuncMap(t *testing.T) {
	t.Parallel()

	t.Run("contributes the shared list helpers", func(t *testing.T) {
		t.Parallel()
		testkit.Assert(t, builder.GoFuncMap()).IsNotEmpty("the adapter contributes the shared helpers")
	})

	t.Run("registers every entry under the plugin's own prefix", func(t *testing.T) {
		t.Parallel()
		for name := range builder.GoFuncMap() {
			testkit.HasPrefix(t, name, builder.Name, "every funcmap entry is namespaced")
		}
	})
}

// hasTemplate reports whether name resolves against the adapter's embedded
// tree. A missing template renders nothing and fails nowhere, so presence is
// the whole assertion.
func hasTemplate(t *testing.T, name string) bool {
	t.Helper()
	tree, ok := builder.GoTemplates()
	if !ok {
		t.Fatal("GoTemplates reported no tree")
	}
	_, err := fs.Stat(tree, name)
	return err == nil
}
