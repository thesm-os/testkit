// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package scope_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all"
	"go.thesmos.sh/testkit/generator/spec/scope"
)

func TestScope(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.Scope)) > 0,
			"scope consumer registered")
	})

	t.Run("Has/Get reflect attached payload", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, scope.Has(&m), "absent without attachment")
		_, ok := scope.Get(&m)
		testkit.False(t, ok, "Get returns ok=false when absent")

		spec.Set(&m.Attachments, directive.Scope, scope.Payload{Name: "billing.read"})
		testkit.True(t, scope.Has(&m), "present after Set")
		got, ok := scope.Get(&m)
		testkit.True(t, ok, "Get returns ok=true when present")
		testkit.Equal(t, got.Name, "billing.read", "scope name round-trips")
	})

	t.Run("Enrich/consume stores scope name", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Svc",
				Methods: []generator.MethodInfo{{Name: "Charge"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Charge",
						Directives: []directive.Directive{
							{Name: directive.Scope, Args: []string{"billing.write"}},
						},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.NoError(t, err, "enrich succeeds")
		got, ok := scope.Get(&data.Methods[0])
		testkit.True(t, ok, "payload attached")
		testkit.Equal(t, got.Name, "billing.write", "scope name stored")
	})

	t.Run("Enrich/consume rejects empty args", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Svc",
				Methods: []generator.MethodInfo{{Name: "Charge"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name:       "Charge",
						Directives: []directive.Directive{{Name: directive.Scope}},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.Error(t, err, "empty args")
	})
}
