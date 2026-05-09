// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package pagination_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all"
	"go.thesmos.sh/testkit/generator/spec/pagination"
)

func TestPagination(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.Pagination)) > 0,
			"pagination consumer registered")
	})

	t.Run("Has/Get reflect attached payload", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, pagination.Has(&m), "absent without attachment")
		_, ok := pagination.Get(&m)
		testkit.False(t, ok, "Get returns ok=false when absent")

		spec.Set(&m.Attachments, directive.Pagination, pagination.Payload{CursorField: "NextPage"})
		testkit.True(t, pagination.Has(&m), "present after Set")
		got, ok := pagination.Get(&m)
		testkit.True(t, ok, "Get returns ok=true when present")
		testkit.Equal(t, got.CursorField, "NextPage", "cursor field round-trips")
	})

	t.Run("Enrich/consume stores cursor field", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Store",
				Methods: []generator.MethodInfo{{Name: "List"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "List",
						Directives: []directive.Directive{
							{Name: directive.Pagination, Args: []string{"NextPage"}},
						},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.NoError(t, err, "enrich succeeds")
		got, ok := pagination.Get(&data.Methods[0])
		testkit.True(t, ok, "payload attached")
		testkit.Equal(t, got.CursorField, "NextPage", "cursor field stored")
	})

	t.Run("Enrich/consume rejects empty args", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Store",
				Methods: []generator.MethodInfo{{Name: "List"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name:       "List",
						Directives: []directive.Directive{{Name: directive.Pagination}},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.Error(t, err, "empty args")
	})
}
