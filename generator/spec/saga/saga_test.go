// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package saga_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all"
	"go.thesmos.sh/testkit/generator/spec/saga"
)

func TestSaga(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.Saga)) > 0, "saga consumer registered")
	})

	t.Run("Has/Get reflect attached payload", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, saga.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.Saga, saga.Payload{Steps: []string{"A", "B"}})
		got, ok := saga.Get(&m)
		testkit.True(t, ok, "Get returns ok=true when present")
		testkit.Equal(t, got.Steps, []string{"A", "B"}, "steps round-trip")
	})

	t.Run("Enrich resolves every step", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Order",
				Methods: []generator.MethodInfo{{Name: "Run"}, {Name: "S1"}, {Name: "S2"}, {Name: "S3"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Run",
						Directives: []directive.Directive{
							{Name: directive.Saga, Args: []string{"S1", "S2", "S3"}},
						},
					},
				},
			},
		}
		testkit.NoError(t, spec.Enrich(data, nil), "enrich succeeds")
		got, _ := saga.Get(&data.Methods[0])
		testkit.Equal(t, got.Steps, []string{"S1", "S2", "S3"}, "all steps resolved in order")
	})

	t.Run("Enrich rejects unknown step", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Order",
				Methods: []generator.MethodInfo{{Name: "Run"}, {Name: "S1"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Run",
						Directives: []directive.Directive{
							{Name: directive.Saga, Args: []string{"S1", "Missing"}},
						},
					},
				},
			},
		}
		testkit.Error(t, spec.Enrich(data, nil), "unknown step")
	})

	t.Run("Enrich rejects empty step list", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Order",
				Methods: []generator.MethodInfo{{Name: "Run"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Run",
						Directives: []directive.Directive{
							{Name: directive.Saga},
						},
					},
				},
			},
		}
		testkit.Error(t, spec.Enrich(data, nil), "needs at least one step")
	})
}
