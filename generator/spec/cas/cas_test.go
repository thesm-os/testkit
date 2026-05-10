// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cas_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all"
	"go.thesmos.sh/testkit/generator/spec/cas"
)

func TestCAS(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.CAS)) > 0, "cas consumer registered")
	})

	t.Run("Has/Get reflect attached payload", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, cas.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.CAS, cas.Payload{VersionField: "Version"})
		got, ok := cas.Get(&m)
		testkit.True(t, ok, "Get returns ok=true when present")
		testkit.Equal(t, got.VersionField, "Version", "round-trip")
	})

	t.Run("Enrich attaches the version field name", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Store",
				Methods: []generator.MethodInfo{{Name: "Put"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Put",
						Directives: []directive.Directive{
							{Name: directive.CAS, Args: []string{"Version"}},
						},
					},
				},
			},
		}
		testkit.NoError(t, spec.Enrich(data, nil), "enrich succeeds")
		got, _ := cas.Get(&data.Methods[0])
		testkit.Equal(t, got.VersionField, "Version", "version field attached")
	})

	t.Run("Enrich rejects missing arg", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Store",
				Methods: []generator.MethodInfo{{Name: "Put"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Put",
						Directives: []directive.Directive{
							{Name: directive.CAS},
						},
					},
				},
			},
		}
		testkit.Error(t, spec.Enrich(data, nil), "missing required arg")
	})
}
