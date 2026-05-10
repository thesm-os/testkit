// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package pool_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all"
	"go.thesmos.sh/testkit/generator/spec/pool"
)

func TestPool(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.Pool)) > 0, "pool consumer registered")
	})

	t.Run("Has/Get reflect attached payload", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, pool.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.Pool, pool.Payload{Put: "Put"})
		got, ok := pool.Get(&m)
		testkit.True(t, ok, "Get returns ok=true when present")
		testkit.Equal(t, got.Put, "Put", "round-trip")
	})

	t.Run("Enrich resolves put sibling", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Pool",
				Methods: []generator.MethodInfo{{Name: "Get"}, {Name: "Put"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Get",
						Directives: []directive.Directive{
							{Name: directive.Pool, Args: []string{"Put"}},
						},
					},
				},
			},
		}
		testkit.NoError(t, spec.Enrich(data, nil), "enrich succeeds")
		got, _ := pool.Get(&data.Methods[0])
		testkit.Equal(t, got.Put, "Put", "put resolved")
	})

	t.Run("Enrich rejects unknown put", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Pool",
				Methods: []generator.MethodInfo{{Name: "Get"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Get",
						Directives: []directive.Directive{
							{Name: directive.Pool, Args: []string{"Missing"}},
						},
					},
				},
			},
		}
		testkit.Error(t, spec.Enrich(data, nil), "unknown put")
	})
}
