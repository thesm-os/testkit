// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package validates_test

import (
	"go/types"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all"
	"go.thesmos.sh/testkit/generator/spec/validates"
)

// errorSig returns a *types.Signature with a single error result.
func errorSig() *types.Signature {
	errIface := types.Universe.Lookup("error").Type()
	results := types.NewTuple(types.NewVar(0, nil, "", errIface))
	return types.NewSignatureType(nil, nil, nil, nil, results, false)
}

// noErrorSig returns a *types.Signature with no results.
func noErrorSig() *types.Signature {
	return types.NewSignatureType(nil, nil, nil, nil, nil, false)
}

func TestValidates(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.Validates)) > 0,
			"validates consumer registered")
	})

	t.Run("Has/Get reflect attached payload", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, validates.Has(&m), "absent without attachment")
		_, ok := validates.Get(&m)
		testkit.False(t, ok, "Get returns ok=false when absent")

		spec.Set(&m.Attachments, directive.Validates, validates.Payload{Field: "ID"})
		testkit.True(t, validates.Has(&m), "present after Set")
		got, ok := validates.Get(&m)
		testkit.True(t, ok, "Get returns ok=true when present")
		testkit.Equal(t, got.Field, "ID", "field name round-trips")
	})

	t.Run("Enrich/consume stores field name", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Store",
				Methods: []generator.MethodInfo{{Name: "Create", Signature: errorSig()}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name:      "Create",
						Signature: errorSig(),
						Directives: []directive.Directive{
							{Name: directive.Validates, Args: []string{"REQ-001"}},
						},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.NoError(t, err, "enrich succeeds")
		got, ok := validates.Get(&data.Methods[0])
		testkit.True(t, ok, "payload attached")
		testkit.Equal(t, got.Field, "REQ-001", "field stored")
	})

	t.Run("Enrich/consume rejects empty args", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Store",
				Methods: []generator.MethodInfo{{Name: "Create", Signature: errorSig()}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name:       "Create",
						Signature:  errorSig(),
						Directives: []directive.Directive{{Name: directive.Validates}},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.Error(t, err, "empty args")
	})

	t.Run("Enrich/consume rejects method without error return", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "Store",
				Methods: []generator.MethodInfo{{Name: "Create", Signature: noErrorSig()}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name:      "Create",
						Signature: noErrorSig(),
						Directives: []directive.Directive{
							{Name: directive.Validates, Args: []string{"ID"}},
						},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.Error(t, err, "no error return")
	})
}
