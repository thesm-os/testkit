// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sentinel_test

import (
	"io/fs"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/sentinel"
)

// The Go adapter's output set is what Layout appends to the anchor's basename,
// so the suffix is a naming contract with every consumer that has run the
// generator.
func TestGoOutputs(t *testing.T) {
	t.Parallel()

	t.Run("declares a single output", func(t *testing.T) {
		t.Parallel()
		// This plugin generates no production code, so there is no second half
		// to route separately the way the double and the builder have.
		testkit.Len(t, sentinel.GoOutputs(), 1, "the adapter declares one file")
	})

	t.Run("marks generated files so tooling can skip them", func(t *testing.T) {
		t.Parallel()
		testkit.Contains(t, sentinel.GoSuffix, ".gen", "the output is marked generated")
	})

	t.Run("ends in _test.go so the package shift fires", func(t *testing.T) {
		t.Parallel()
		// The framework keys the external-test-package shift off this exact
		// ending. A suffix that merely looked test-ish would land the checks in
		// the package under test, where they could reach unexported state and
		// stop proving what a consumer can see.
		testkit.HasSuffix(t, sentinel.GoSuffix, "_test.go", "the output triggers the shift")
	})
}

// A template the backend cannot find renders nothing and fails nowhere, so the
// entry point is checked by name rather than assumed present.
func TestGoTemplates(t *testing.T) {
	t.Parallel()

	t.Run("carries the template the checks render from", func(t *testing.T) {
		t.Parallel()
		tree, ok := sentinel.GoTemplates()
		testkit.True(t, ok, "the adapter reports a tree")
		_, err := fs.Stat(tree, "sentinel.tests.tmpl")
		testkit.NoError(t, err, "the template must ship")
	})
}

// The funcmap is where this plugin and its neighbours must not collide: the
// backend fails the whole run when two register one extension name.
func TestGoFuncMap(t *testing.T) {
	t.Parallel()

	t.Run("contributes the shared helpers", func(t *testing.T) {
		t.Parallel()
		testkit.Assert(t, sentinel.GoFuncMap()).IsNotEmpty("the adapter contributes helpers")
	})

	t.Run("registers every entry under the plugin's own prefix", func(t *testing.T) {
		t.Parallel()
		for name := range sentinel.GoFuncMap() {
			testkit.HasPrefix(t, name, sentinel.Name, "every funcmap entry is namespaced")
		}
	})
}
