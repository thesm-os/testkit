// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package acquire_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/acquire"
	_ "go.thesmos.sh/testkit/generator/spec/all"
)

func TestAcquire(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.Acquire)) > 0, "acquire consumer registered")
	})

	t.Run("Has/Get reflect attached payload", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, acquire.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.Acquire, acquire.Payload{Release: "Release"})
		got, ok := acquire.Get(&m)
		testkit.True(t, ok, "Get returns ok=true when present")
		testkit.Equal(t, got.Release, "Release", "round-trip")
	})

	t.Run("Enrich resolves release sibling", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Lock",
				Methods: []generator.MethodInfo{{Name: "Acquire"}, {Name: "Release"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Acquire",
						Directives: []directive.Directive{
							{Name: directive.Acquire, Args: []string{"Release"}},
						},
					},
				},
			},
		}
		testkit.NoError(t, spec.Enrich(data, nil), "enrich succeeds")
		got, _ := acquire.Get(&data.Methods[0])
		testkit.Equal(t, got.Release, "Release", "release resolved")
	})

	t.Run("Enrich rejects unknown release", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Lock",
				Methods: []generator.MethodInfo{{Name: "Acquire"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Acquire",
						Directives: []directive.Directive{
							{Name: directive.Acquire, Args: []string{"Missing"}},
						},
					},
				},
			},
		}
		testkit.Error(t, spec.Enrich(data, nil), "unknown release")
	})
}
