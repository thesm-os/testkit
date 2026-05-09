// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package spec_test

import (
	"go/types"
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

	t.Run("NonCtxParamCount + NonCtxParamAt skip leading ctx", func(t *testing.T) {
		t.Parallel()
		// Use spec.Analyze against testdata/basic's Sampler so the
		// signatures are real go/types objects with HasContext()
		// returning the correct value.
		pkg, err := generator.NewLoader().Load("./../testdata/basic", "")
		testkit.NoError(t, err, "Load testdata/basic")
		data, err := spec.Analyze(pkg, []string{"Sampler"},
			generator.DefaultConfig(), generator.Options{Output: "samplertest/x.gen.go"})
		testkit.NoError(t, err, "Analyze")
		byName := make(map[string]spec.Method, len(data.Methods))
		for _, m := range data.Methods {
			byName[m.Name] = m
		}
		// Apply(ctx, key string, item Item) error → 2 non-ctx params.
		apply := byName["Apply"]
		testkit.Equal(t, apply.NonCtxParamCount(), 2, "Apply has 2 non-ctx params")
		testkit.Equal(t, apply.NonCtxParamAt(0).String(), "string", "param 0 is string")
		testkit.Equal(t, apply.NonCtxParamAt(1).String(),
			"go.thesmos.sh/testkit/generator/testdata/basic.Item",
			"param 1 is basic.Item")
	})

	t.Run("NonCtxParamCount returns NumParams when no ctx is present", func(t *testing.T) {
		t.Parallel()
		// Build a synthetic Method with no context to exercise the
		// false branch of HasContext().
		var noCtxSig types.Type = types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(0, nil, "x", types.Typ[types.String])),
			nil, false)
		m := spec.Method{MethodInfo: generator.MethodInfo{
			Signature: noCtxSig.(*types.Signature),
		}}
		testkit.Equal(t, m.NonCtxParamCount(), 1, "all params count when no ctx")
		testkit.Equal(t, m.NonCtxParamAt(0).String(), "string", "first param is x")
	})
}
