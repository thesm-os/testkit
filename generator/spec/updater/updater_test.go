// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package updater_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all"
	"go.thesmos.sh/testkit/generator/spec/updater"
)

func TestUpdater(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.Updater)) > 0, "updater consumer registered")
	})

	t.Run("Has/Get reflect attached payload", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, updater.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.Updater, updater.Payload{Reader: "Get"})
		got, ok := updater.Get(&m)
		testkit.True(t, ok, "Get returns ok=true when present")
		testkit.Equal(t, got.Reader, "Get", "round-trip")
	})

	t.Run("Enrich resolves Reader sibling", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Store",
				Methods: []generator.MethodInfo{{Name: "Update"}, {Name: "Get"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Update",
						Directives: []directive.Directive{
							{Name: directive.Updater, Args: []string{"Get"}},
						},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.NoError(t, err, "enrich succeeds")
		got, _ := updater.Get(&data.Methods[0])
		testkit.Equal(t, got.Reader, "Get", "reader resolved")
	})

	t.Run("Enrich rejects unknown sibling", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Store",
				Methods: []generator.MethodInfo{{Name: "Update"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Update",
						Directives: []directive.Directive{
							{Name: directive.Updater, Args: []string{"Missing"}},
						},
					},
				},
			},
		}
		testkit.Error(t, spec.Enrich(data, nil), "unknown sibling")
	})
}
