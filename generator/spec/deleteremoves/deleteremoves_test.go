// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package deleteremoves_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all"
	"go.thesmos.sh/testkit/generator/spec/deleteremoves"
)

func TestDeleteRemoves(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.DeleteRemoves)) > 0,
			"delete-removes consumer registered")
	})

	t.Run("Has/Get reflect attached payload", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, deleteremoves.Has(&m), "absent without attachment")
		_, ok := deleteremoves.Get(&m)
		testkit.False(t, ok, "Get returns ok=false when absent")

		spec.Set(&m.Attachments, directive.DeleteRemoves, deleteremoves.Payload{Reader: "Get"})
		testkit.True(t, deleteremoves.Has(&m), "present after Set")
		got, ok := deleteremoves.Get(&m)
		testkit.True(t, ok, "Get returns ok=true when present")
		testkit.Equal(t, got.Reader, "Get", "reader method round-trips")
	})

	t.Run("Enrich/consume resolves reader method", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Store",
				Methods: []generator.MethodInfo{{Name: "Get"}, {Name: "Delete"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Delete",
						Directives: []directive.Directive{
							{Name: directive.DeleteRemoves, Args: []string{"Get"}},
						},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.NoError(t, err, "enrich succeeds")
		got, ok := deleteremoves.Get(&data.Methods[0])
		testkit.True(t, ok, "payload attached")
		testkit.Equal(t, got.Reader, "Get", "reader resolved")
	})

	t.Run("Enrich/consume rejects unknown method", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Store",
				Methods: []generator.MethodInfo{{Name: "Delete"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Delete",
						Directives: []directive.Directive{
							{Name: directive.DeleteRemoves, Args: []string{"Missing"}},
						},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.Error(t, err, "unknown reader method")
	})
}
