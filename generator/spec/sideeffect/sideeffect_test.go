// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sideeffect_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all"
	"go.thesmos.sh/testkit/generator/spec/sideeffect"
)

func TestSideEffect(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.SideEffect)) > 0,
			"sideeffect consumer registered")
	})

	t.Run("Has/Get reflect attached payload", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, sideeffect.Has(&m), "absent without attachment")
		_, ok := sideeffect.Get(&m)
		testkit.False(t, ok, "Get returns ok=false when absent")

		spec.Set(&m.Attachments, directive.SideEffect, sideeffect.Payload{Method: "Get"})
		testkit.True(t, sideeffect.Has(&m), "present after Set")
		got, ok := sideeffect.Get(&m)
		testkit.True(t, ok, "Get returns ok=true when present")
		testkit.Equal(t, got.Method, "Get", "method name round-trips")
	})

	t.Run("Enrich/consume resolves observation method", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Svc",
				Methods: []generator.MethodInfo{{Name: "Audit"}, {Name: "Write"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Write",
						Directives: []directive.Directive{
							{Name: directive.SideEffect, Args: []string{"Audit"}},
						},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.NoError(t, err, "enrich succeeds")
		got, ok := sideeffect.Get(&data.Methods[0])
		testkit.True(t, ok, "payload attached")
		testkit.Equal(t, got.Method, "Audit", "observation method resolved")
	})

	t.Run("Enrich/consume rejects unknown method", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Svc",
				Methods: []generator.MethodInfo{{Name: "Write"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Write",
						Directives: []directive.Directive{
							{Name: directive.SideEffect, Args: []string{"Missing"}},
						},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.Error(t, err, "unknown method")
	})
}
