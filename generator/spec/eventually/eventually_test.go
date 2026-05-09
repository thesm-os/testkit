// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package eventually_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all"
	"go.thesmos.sh/testkit/generator/spec/eventually"
)

func TestEventually(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.Eventually)) > 0,
			"eventually consumer registered")
	})

	t.Run("Has/Get reflect attached payload", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, eventually.Has(&m), "absent without attachment")
		_, ok := eventually.Get(&m)
		testkit.False(t, ok, "Get returns ok=false when absent")

		spec.Set(&m.Attachments, directive.Eventually, eventually.Payload{Duration: "500ms"})
		testkit.True(t, eventually.Has(&m), "present after Set")
		got, ok := eventually.Get(&m)
		testkit.True(t, ok, "Get returns ok=true when present")
		testkit.Equal(t, got.Duration, "500ms", "duration round-trips")
	})

	t.Run("Enrich/consume stores duration", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Svc",
				Methods: []generator.MethodInfo{{Name: "Ready"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Ready",
						Directives: []directive.Directive{
							{Name: directive.Eventually, Args: []string{"500ms"}},
						},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.NoError(t, err, "enrich succeeds")
		got, ok := eventually.Get(&data.Methods[0])
		testkit.True(t, ok, "payload attached")
		testkit.Equal(t, got.Duration, "500ms", "duration stored")
	})

	t.Run("Enrich/consume rejects empty args", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Svc",
				Methods: []generator.MethodInfo{{Name: "Ready"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name:       "Ready",
						Directives: []directive.Directive{{Name: directive.Eventually}},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.Error(t, err, "empty args")
	})
}
