// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package upserter_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all"
	"go.thesmos.sh/testkit/generator/spec/upserter"
)

func TestUpserter(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.Upserter)) > 0, "upserter consumer registered")
	})

	t.Run("Has/Get reflect attached payload", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, upserter.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.Upserter, upserter.Payload{Reader: "Get"})
		got, ok := upserter.Get(&m)
		testkit.True(t, ok, "Get returns ok=true when present")
		testkit.Equal(t, got.Reader, "Get", "round-trip")
	})

	t.Run("Enrich resolves Reader sibling", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Store",
				Methods: []generator.MethodInfo{{Name: "Upsert"}, {Name: "Get"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Upsert",
						Directives: []directive.Directive{
							{Name: directive.Upserter, Args: []string{"Get"}},
						},
					},
				},
			},
		}
		testkit.NoError(t, spec.Enrich(data, nil), "enrich succeeds")
		got, _ := upserter.Get(&data.Methods[0])
		testkit.Equal(t, got.Reader, "Get", "reader resolved")
	})

	t.Run("Enrich rejects unknown sibling", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Store",
				Methods: []generator.MethodInfo{{Name: "Upsert"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Upsert",
						Directives: []directive.Directive{
							{Name: directive.Upserter, Args: []string{"Missing"}},
						},
					},
				},
			},
		}
		testkit.Error(t, spec.Enrich(data, nil), "unknown sibling")
	})
}
