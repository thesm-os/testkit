// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package watcher_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all"
	"go.thesmos.sh/testkit/generator/spec/watcher"
)

func TestWatcher(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.Watcher)) > 0, "watcher consumer registered")
	})

	t.Run("Has/Get reflect attached payload", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, watcher.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.Watcher, watcher.Payload{Trigger: "Set"})
		got, ok := watcher.Get(&m)
		testkit.True(t, ok, "Get returns ok=true when present")
		testkit.Equal(t, got.Trigger, "Set", "round-trip")
	})

	t.Run("Enrich resolves trigger sibling", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Config",
				Methods: []generator.MethodInfo{{Name: "Watch"}, {Name: "Set"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Watch",
						Directives: []directive.Directive{
							{Name: directive.Watcher, Args: []string{"Set"}},
						},
					},
				},
			},
		}
		testkit.NoError(t, spec.Enrich(data, nil), "enrich succeeds")
		got, _ := watcher.Get(&data.Methods[0])
		testkit.Equal(t, got.Trigger, "Set", "trigger resolved")
	})

	t.Run("Enrich rejects unknown trigger", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Config",
				Methods: []generator.MethodInfo{{Name: "Watch"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Watch",
						Directives: []directive.Directive{
							{Name: directive.Watcher, Args: []string{"Missing"}},
						},
					},
				},
			},
		}
		testkit.Error(t, spec.Enrich(data, nil), "unknown trigger")
	})
}
