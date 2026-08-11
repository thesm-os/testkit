// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package fault_test

import (
	"io/fs"
	"testing"

	"go.thesmos.sh/eidos/lang/golang"
	sdkgolang "go.thesmos.sh/eidos/sdk/golang"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/fault"
	"go.thesmos.sh/testkit/generator/stub"
)

// The output set is not this plugin's own naming choice — it is the mechanism
// by which the contribution reaches the double's file at all. Layout composes
// a contributor's Target from the contributing plugin's suffix table, so a
// suffix that drifted from the stub generator's would silently write a second
// file instead of failing.
func TestGoOutputs(t *testing.T) {
	t.Parallel()

	t.Run("declares the same pair the host declares", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, fault.GoOutputs(), stub.GoOutputs(),
			"the contribution routes with the double or not at all")
	})
}

// A template the backend cannot find renders nothing and fails nowhere, so
// both entry points are checked by name rather than assumed present.
func TestGoTemplates(t *testing.T) {
	t.Parallel()

	t.Run("reports a tree", func(t *testing.T) {
		t.Parallel()
		_, ok := fault.New().Templates(golang.Language)
		testkit.True(t, ok, "the embedded tree is always available")
	})

	t.Run("carries the template the helpers render from", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, hasTemplate(t, "fault.helpers.tmpl"), "the helpers template must ship")
	})

	t.Run("carries the template the checks render from", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, hasTemplate(t, "fault.test.tmpl"), "the companion template must ship")
	})
}

// The funcmap is where this plugin and its host must not collide: the backend
// fails the whole run when two plugins register one extension name.
func TestGoFuncMap(t *testing.T) {
	t.Parallel()

	t.Run("contributes the shared list helpers", func(t *testing.T) {
		t.Parallel()
		funcs := fault.New().TemplateFuncs(golang.Language)
		testkit.Assert(t, funcs).IsNotEmpty("the adapter contributes the shared Go helpers")
	})

	t.Run("shares no entry with the host it contributes into", func(t *testing.T) {
		t.Parallel()
		host := stub.New().TemplateFuncs(golang.Language)
		for name := range fault.New().TemplateFuncs(golang.Language) {
			_, clash := host[name]
			testkit.False(t, clash, "the two plugins must not both register "+name)
		}
	})

	t.Run("registers no replacement for the backend's external builtin", func(t *testing.T) {
		t.Parallel()
		// `ref` is deliberately absent. It resolved a short alias against a
		// private table, and the backend's own `external "<path>" "<Name>"`
		// builtin already does exactly that — so the templates call the builtin
		// and this plugin registers nothing in its place.
		//
		// That every name the templates *do* call resolves is asserted over the
		// whole plugin set, in plugins_test.go, by reading the call sites from
		// the parser. It is not part of [plugintest.RunSuite], which parses each
		// template with every unresolved name stubbed so that it judges syntax
		// alone — so a hand-kept list here would have been the only thing
		// standing between a template and a render failure in a consumer's
		// build, and a hand-kept list drifts in the direction that ships.
		funcs := fault.New().TemplateFuncs(golang.Language)
		_, custom := funcs[fault.Name+sdkgolang.FuncPrefixSeparator+"ref"]
		testkit.False(t, custom, "ref is the backend's external builtin, not a plugin helper")
	})
}

// hasTemplate reports whether the named template ships in the plugin's Go tree.
//
// Presence is the whole assertion, and it is worth making: a template the
// backend cannot find renders nothing and fails nowhere, so the output simply
// comes out short.
//
// Looked up by base name because that is how the backend registers them: a
// tree rooted one directory too high still resolves the file by a longer path
// and still renders nothing.
func hasTemplate(t *testing.T, name string) bool {
	t.Helper()
	tree, ok := fault.New().Templates(golang.Language)
	if !ok {
		t.Fatal("the plugin reports no Go template tree")
	}
	_, err := fs.Stat(tree, name)
	return err == nil
}
