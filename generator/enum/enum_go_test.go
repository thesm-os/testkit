// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enum_test

import (
	"io/fs"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/enum"
)

// The Go adapter's output set is what Layout appends to the source basename,
// so the suffixes are a naming contract with every consumer that has run the
// generator.
func TestGoOutputs(t *testing.T) {
	t.Parallel()

	t.Run("declares the API and its checks", func(t *testing.T) {
		t.Parallel()
		testkit.Len(t, enum.GoOutputs(), 2, "the adapter declares a pair")
	})

	t.Run("puts the primary first", func(t *testing.T) {
		t.Parallel()
		// The framework requires the empty-tag entry at index 0, and rejects
		// the whole plugin otherwise.
		testkit.Equal(t, enum.GoOutputs()[0].Tag, "", "the primary leads")
	})

	t.Run("marks generated files so tooling can skip them", func(t *testing.T) {
		t.Parallel()
		testkit.Contains(t, enum.GoPrimarySuffix, ".enum_gen", "the API is marked generated")
	})

	t.Run("ends the checks in _test.go so the package shift fires", func(t *testing.T) {
		t.Parallel()
		// The framework keys the external-test-package shift off this exact
		// ending. Without it the checks would land in the enum's own package,
		// where they would no longer drive it the way a consumer does.
		testkit.HasSuffix(t, enum.GoTestSuffix, "_test.go", "the companion triggers the shift")
	})
}

// A template the backend cannot find renders nothing and fails nowhere, so both
// entry points are checked by name rather than assumed present.
func TestGoTemplates(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"enum.api.tmpl", "enum.test.tmpl"} {
		t.Run("carries "+name, func(t *testing.T) {
			t.Parallel()
			tree, ok := enum.GoTemplates()
			testkit.True(t, ok, "the adapter reports a tree")
			_, err := fs.Stat(tree, name)
			testkit.NoError(t, err, name+" must ship")
		})
	}
}

// The funcmap is where this plugin and its neighbours must not collide: the
// backend fails the whole run when two register one extension name.
func TestGoFuncMap(t *testing.T) {
	t.Parallel()

	t.Run("contributes the shared helpers", func(t *testing.T) {
		t.Parallel()
		testkit.Assert(t, enum.GoFuncMap()).IsNotEmpty("the adapter contributes helpers")
	})

	t.Run("registers every entry under the plugin's own prefix", func(t *testing.T) {
		t.Parallel()
		for name := range enum.GoFuncMap() {
			testkit.HasPrefix(t, name, enum.Name, "every funcmap entry is namespaced")
		}
	})
}
