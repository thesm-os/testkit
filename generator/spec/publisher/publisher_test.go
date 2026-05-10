// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package publisher_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all"
	"go.thesmos.sh/testkit/generator/spec/publisher"
)

func TestPublisher(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.Publisher)) > 0, "publisher consumer registered")
	})

	t.Run("Has/Get reflect attached payload", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, publisher.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.Publisher, publisher.Payload{Subscribe: "Sub"})
		got, ok := publisher.Get(&m)
		testkit.True(t, ok, "Get returns ok=true when present")
		testkit.Equal(t, got.Subscribe, "Sub", "round-trip")
	})

	t.Run("Enrich resolves subscribe sibling", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Bus",
				Methods: []generator.MethodInfo{{Name: "Publish"}, {Name: "Subscribe"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Publish",
						Directives: []directive.Directive{
							{Name: directive.Publisher, Args: []string{"Subscribe"}},
						},
					},
				},
			},
		}
		testkit.NoError(t, spec.Enrich(data, nil), "enrich succeeds")
		got, _ := publisher.Get(&data.Methods[0])
		testkit.Equal(t, got.Subscribe, "Subscribe", "subscribe resolved")
	})

	t.Run("Enrich rejects unknown subscribe", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Bus",
				Methods: []generator.MethodInfo{{Name: "Publish"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Publish",
						Directives: []directive.Directive{
							{Name: directive.Publisher, Args: []string{"Missing"}},
						},
					},
				},
			},
		}
		testkit.Error(t, spec.Enrich(data, nil), "unknown subscribe")
	})
}
