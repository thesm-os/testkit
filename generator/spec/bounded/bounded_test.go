// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bounded_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all"
	"go.thesmos.sh/testkit/generator/spec/bounded"
)

func TestBounded(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.Bounded)) > 0,
			"bounded consumer registered")
	})

	t.Run("Has/Get reflect attached payload", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, bounded.Has(&m), "absent without attachment")
		_, ok := bounded.Get(&m)
		testkit.False(t, ok, "Get returns ok=false when absent")

		spec.Set(&m.Attachments, directive.Bounded, bounded.Payload{Min: "0", Max: "100"})
		testkit.True(t, bounded.Has(&m), "present after Set")
		got, ok := bounded.Get(&m)
		testkit.True(t, ok, "Get returns ok=true when present")
		testkit.Equal(t, got.Min, "0", "min round-trips")
		testkit.Equal(t, got.Max, "100", "max round-trips")
	})

	t.Run("Enrich/consume parses min..max", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Iface",
				Methods: []generator.MethodInfo{{Name: "Score"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Score",
						Directives: []directive.Directive{
							{Name: directive.Bounded, Args: []string{"0..100"}},
						},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.NoError(t, err, "enrich succeeds")
		got, ok := bounded.Get(&data.Methods[0])
		testkit.True(t, ok, "payload attached")
		testkit.Equal(t, got.Min, "0", "min parsed")
		testkit.Equal(t, got.Max, "100", "max parsed")
	})

	t.Run("Enrich/consume rejects missing args", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Iface",
				Methods: []generator.MethodInfo{{Name: "Score"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name:       "Score",
						Directives: []directive.Directive{{Name: directive.Bounded}},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.Error(t, err, "missing args")
	})

	t.Run("Enrich/consume rejects missing separator", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Iface",
				Methods: []generator.MethodInfo{{Name: "Score"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Score",
						Directives: []directive.Directive{
							{Name: directive.Bounded, Args: []string{"100"}},
						},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.Error(t, err, "no .. separator")
	})

	t.Run("Enrich/consume rejects empty bounds", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Iface",
				Methods: []generator.MethodInfo{{Name: "Score"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Score",
						Directives: []directive.Directive{
							{Name: directive.Bounded, Args: []string{"..100"}},
						},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.Error(t, err, "empty lower bound")
	})
}
