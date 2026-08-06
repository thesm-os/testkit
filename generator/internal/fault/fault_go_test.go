// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package fault_test

import (
	"io/fs"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/fault"
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
		_, ok := fault.GoTemplates()
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
		testkit.Assert(t, fault.GoFuncMap()).IsNotEmpty("the adapter contributes the shared Go helpers")
	})

	t.Run("shares no entry with the host it contributes into", func(t *testing.T) {
		t.Parallel()
		host := stub.GoFuncMap()
		for name := range fault.GoFuncMap() {
			_, clash := host[name]
			testkit.False(t, clash, "the two plugins must not both register "+name)
		}
	})
}

// hasTemplate reports whether name resolves against the adapter's embedded
// tree. A missing template renders nothing and fails nowhere, so presence is
// the whole assertion — the stat error itself carries nothing a reader needs.
func hasTemplate(t *testing.T, name string) bool {
	t.Helper()
	tree, ok := fault.GoTemplates()
	if !ok {
		t.Fatal("GoTemplates reported no tree")
	}
	_, err := fs.Stat(tree, name)
	return err == nil
}
