// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cursor_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all"
	"go.thesmos.sh/testkit/generator/spec/cursor"
)

func TestCursor(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.Cursor)) > 0, "cursor consumer registered")
	})

	t.Run("Has/Get reflect attached payload", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, cursor.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.Cursor, cursor.Payload{Close: "Close"})
		got, ok := cursor.Get(&m)
		testkit.True(t, ok, "Get returns ok=true when present")
		testkit.Equal(t, got.Close, "Close", "round-trip")
	})

	t.Run("Enrich resolves close sibling", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Iter",
				Methods: []generator.MethodInfo{{Name: "Next"}, {Name: "Close"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Next",
						Directives: []directive.Directive{
							{Name: directive.Cursor, Args: []string{"Close"}},
						},
					},
				},
			},
		}
		testkit.NoError(t, spec.Enrich(data, nil), "enrich succeeds")
		got, _ := cursor.Get(&data.Methods[0])
		testkit.Equal(t, got.Close, "Close", "close resolved")
	})

	t.Run("Enrich rejects unknown close", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Iter",
				Methods: []generator.MethodInfo{{Name: "Next"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Next",
						Directives: []directive.Directive{
							{Name: directive.Cursor, Args: []string{"Missing"}},
						},
					},
				},
			},
		}
		testkit.Error(t, spec.Enrich(data, nil), "unknown close")
	})
}
