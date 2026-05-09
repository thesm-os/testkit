// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package streamreflectsmutations_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all"
	"go.thesmos.sh/testkit/generator/spec/streamreflectsmutations"
)

func TestStreamReflectsMutations(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.StreamReflectsMutations)) > 0,
			"stream-reflects-mutations consumer registered")
	})

	t.Run("Has/Get reflect attached payload", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, streamreflectsmutations.Has(&m), "absent without attachment")
		_, ok := streamreflectsmutations.Get(&m)
		testkit.False(t, ok, "Get returns ok=false when absent")

		spec.Set(&m.Attachments, directive.StreamReflectsMutations,
			streamreflectsmutations.Payload{Stream: "Scan"})
		testkit.True(t, streamreflectsmutations.Has(&m), "present after Set")
		got, ok := streamreflectsmutations.Get(&m)
		testkit.True(t, ok, "Get returns ok=true when present")
		testkit.Equal(t, got.Stream, "Scan", "stream method round-trips")
	})

	t.Run("Enrich/consume resolves stream method", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Store",
				Methods: []generator.MethodInfo{{Name: "Put"}, {Name: "Scan"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Put",
						Directives: []directive.Directive{
							{Name: directive.StreamReflectsMutations, Args: []string{"Scan"}},
						},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.NoError(t, err, "enrich succeeds")
		got, ok := streamreflectsmutations.Get(&data.Methods[0])
		testkit.True(t, ok, "payload attached")
		testkit.Equal(t, got.Stream, "Scan", "stream method resolved")
	})

	t.Run("Enrich/consume rejects unknown method", func(t *testing.T) {
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
							{Name: directive.StreamReflectsMutations, Args: []string{"Missing"}},
						},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.Error(t, err, "unknown stream method")
	})
}
