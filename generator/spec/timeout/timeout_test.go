// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package timeout_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all"
	"go.thesmos.sh/testkit/generator/spec/timeout"
)

func TestTimeout(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.Timeout)) > 0,
			"timeout consumer registered")
	})

	t.Run("Has/Get reflect attached payload", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, timeout.Has(&m), "absent without attachment")
		_, ok := timeout.Get(&m)
		testkit.False(t, ok, "Get returns ok=false when absent")

		spec.Set(&m.Attachments, directive.Timeout, timeout.Payload{Duration: "100ms"})
		testkit.True(t, timeout.Has(&m), "present after Set")
		got, ok := timeout.Get(&m)
		testkit.True(t, ok, "Get returns ok=true when present")
		testkit.Equal(t, got.Duration, "100ms", "duration round-trips")
	})

	t.Run("Enrich/consume stores duration", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Svc",
				Methods: []generator.MethodInfo{{Name: "Ping"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Ping",
						Directives: []directive.Directive{
							{Name: directive.Timeout, Args: []string{"5s"}},
						},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.NoError(t, err, "enrich succeeds")
		got, ok := timeout.Get(&data.Methods[0])
		testkit.True(t, ok, "payload attached")
		testkit.Equal(t, got.Duration, "5s", "duration stored")
	})

	t.Run("Enrich/consume rejects empty args", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Svc",
				Methods: []generator.MethodInfo{{Name: "Ping"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name:       "Ping",
						Directives: []directive.Directive{{Name: directive.Timeout}},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.Error(t, err, "empty args")
	})
}
