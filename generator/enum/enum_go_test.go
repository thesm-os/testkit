// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enum_test

import (
	"io/fs"
	"maps"
	"slices"
	"testing"

	"go.thesmos.sh/eidos/lang/golang"
	sdkgolang "go.thesmos.sh/eidos/sdk/golang"

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

	t.Run("declares no routable output for another language", func(t *testing.T) {
		t.Parallel()
		// Nil rather than an empty set is what makes Layout report a missing
		// provider instead of composing Go-shaped filenames for a backend that
		// is not Go.
		testkit.Assert(t, enum.New().Outputs("rust")).IsNil("a non-Go backend gets no Go filenames")
	})
}

// A template the backend cannot find renders nothing and fails nowhere, so both
// entry points are checked by name rather than assumed present.
func TestGoTemplates(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"enum.api.tmpl", "enum.test.tmpl"} {
		t.Run("carries "+name, func(t *testing.T) {
			t.Parallel()
			tree, ok := enum.New().Templates(golang.Language)
			testkit.True(t, ok, "the adapter reports a tree")
			_, err := fs.Stat(tree, name)
			testkit.NoError(t, err, name+" must ship")
		})
	}

	t.Run("ships none for another language", func(t *testing.T) {
		t.Parallel()
		_, ok := enum.New().Templates("rust")
		testkit.False(t, ok, "a non-Go backend gets no templates")
	})
}

// The funcmap is where this plugin and its neighbours must not collide: the
// backend fails the whole run when two register one extension name.
func TestGoFuncMap(t *testing.T) {
	t.Parallel()

	t.Run("contributes the shared helpers", func(t *testing.T) {
		t.Parallel()
		testkit.Assert(t, enum.New().TemplateFuncs(golang.Language)).IsNotEmpty("the adapter contributes helpers")
	})

	t.Run("registers every entry under the plugin's own prefix", func(t *testing.T) {
		t.Parallel()
		// The separator too, not just the name: a template calling `enumRef`
		// where the bundle registers `enum_ref` renders nothing and fails
		// nowhere, so the exact prefix is the contract worth pinning.
		for name := range enum.New().TemplateFuncs(golang.Language) {
			testkit.HasPrefix(t, name, enum.Name+sdkgolang.FuncPrefixSeparator,
				"every funcmap entry is namespaced")
		}
	})

	t.Run("registers no helper of its own", func(t *testing.T) {
		t.Parallel()
		// The bundle is exactly the shared one. Everything the templates call
		// is either a backend builtin — `renderExpr`, `external` — or an entry
		// of [golang.AllFuncMap]; a local helper here would be a private copy
		// of Go knowledge the framework already owns, and the last one was:
		// resolving an import alias to a path, which `external` does with the
		// path carried on the emit value.
		got := slices.Sorted(maps.Keys(enum.New().TemplateFuncs(golang.Language)))
		want := slices.Sorted(maps.Keys(golang.AllFuncMap(sdkgolang.FuncPrefix(enum.Name))))
		testkit.Equal(t, got, want, "the plugin adds nothing to the shared bundle")
	})

	t.Run("replaces no backend builtin", func(t *testing.T) {
		t.Parallel()
		// An override changes rendering for every plugin sharing the backend,
		// so the empty return is a deliberate contract rather than an omission.
		testkit.Assert(t, enum.New().TemplateOverrides(golang.Language)).
			IsNil("the plugin replaces no canonical entry")
	})
}

// The runtime path is what a generated check resolves its assertions through.
// Spelled in a template it would have to be corrected in every one of them the
// day the module moves.
func TestGoRuntime(t *testing.T) {
	t.Parallel()

	t.Run("names the module the checks import", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, enum.GoRuntime().Runtime, enum.Module, "the checks reach the module root")
	})
}
