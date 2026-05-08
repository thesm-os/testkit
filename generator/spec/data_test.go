// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package spec_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/shape"
	"go.thesmos.sh/testkit/generator/spec"
)

func TestData(t *testing.T) {
	t.Parallel()

	t.Run("Method embeds MethodInfo and exposes promoted fields", func(t *testing.T) {
		t.Parallel()
		m := spec.Method{
			MethodInfo: generator.MethodInfo{Name: "Get"},
			Shape:      shape.Info{Shape: shape.Reader, KeyType: "string", ValType: "Item"},
		}
		// Promotion check: Name is reachable directly from spec.Method.
		testkit.Equal(t, m.Name, "Get", "MethodInfo promotion")
		testkit.Equal(t, m.Shape.Shape, shape.Reader, "Shape attached")
	})

	t.Run("Attachments map starts nil and tolerates Set", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.True(t, m.Attachments == nil, "Attachments starts nil")
		spec.Set(&m.Attachments, "any", 42)
		testkit.True(t, spec.Has(m.Attachments, "any"), "Set populated the map")
	})

	t.Run("Data composes Package, Interface, Methods, Tracker", func(t *testing.T) {
		t.Parallel()
		// Smoke test: a freshly-allocated Data has the expected zero values
		// and accepts assignment without panic.
		d := &spec.Data{
			Args:    []string{"Store"},
			Tracker: generator.NewImportTracker("p"),
		}
		testkit.Len(t, d.Args, 1, "Args attached")
		testkit.True(t, d.Tracker != nil, "Tracker attached")
		testkit.Equal(t, d.Tracker.LocalPkg(), "p", "Tracker carries local pkg")
	})
}
