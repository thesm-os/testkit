// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package hooks_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all"
	"go.thesmos.sh/testkit/generator/spec/hooks"
)

func TestHooks(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.Hooks)) > 0,
			"hooks consumer registered")
	})

	t.Run("Has/Get reflect attached payload with multiple names", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, hooks.Has(&m), "absent without attachment")
		_, ok := hooks.Get(&m)
		testkit.False(t, ok, "Get returns ok=false when absent")

		spec.Set(&m.Attachments, directive.Hooks, hooks.Payload{
			Names: []string{"BeforeWrite", "AfterWrite"},
		})
		testkit.True(t, hooks.Has(&m), "present after Set")
		got, ok := hooks.Get(&m)
		testkit.True(t, ok, "Get returns ok=true when present")
		testkit.Equal(t, len(got.Names), 2, "captures all names")
		testkit.Equal(t, got.Names[0], "BeforeWrite", "first name")
		testkit.Equal(t, got.Names[1], "AfterWrite", "second name")
	})

	t.Run("Enrich/consume stores hook names", func(t *testing.T) {
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
							{Name: directive.Hooks, Args: []string{"BeforeWrite", "AfterWrite"}},
						},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.NoError(t, err, "enrich succeeds")
		got, ok := hooks.Get(&data.Methods[0])
		testkit.True(t, ok, "payload attached")
		testkit.Equal(t, len(got.Names), 2, "two hook names")
		testkit.Equal(t, got.Names[0], "BeforeWrite", "first hook")
		testkit.Equal(t, got.Names[1], "AfterWrite", "second hook")
	})

	t.Run("Enrich/consume rejects empty args", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Svc",
				Methods: []generator.MethodInfo{{Name: "Write"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name:       "Write",
						Directives: []directive.Directive{{Name: directive.Hooks}},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.Error(t, err, "empty args")
	})
}
