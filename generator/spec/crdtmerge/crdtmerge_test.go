// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package crdtmerge_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all"
	"go.thesmos.sh/testkit/generator/spec/crdtmerge"
)

func TestCRDTMerge(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.CRDTMerge)) > 0,
			"crdt-merge consumer registered")
	})

	t.Run("Has/Get reflect attached payload", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, crdtmerge.Has(&m), "absent without attachment")
		_, ok := crdtmerge.Get(&m)
		testkit.False(t, ok, "Get returns ok=false when absent")

		spec.Set(&m.Attachments, directive.CRDTMerge, crdtmerge.Payload{Other: "Merge"})
		testkit.True(t, crdtmerge.Has(&m), "present after Set")
		got, ok := crdtmerge.Get(&m)
		testkit.True(t, ok, "Get returns ok=true when present")
		testkit.Equal(t, got.Other, "Merge", "counterpart method round-trips")
	})

	t.Run("Enrich/consume resolves counterpart method", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "CRDT",
				Methods: []generator.MethodInfo{{Name: "Merge"}, {Name: "Get"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Merge",
						Directives: []directive.Directive{
							{Name: directive.CRDTMerge, Args: []string{"Merge"}},
						},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.NoError(t, err, "enrich succeeds")
		got, ok := crdtmerge.Get(&data.Methods[0])
		testkit.True(t, ok, "payload attached")
		testkit.Equal(t, got.Other, "Merge", "counterpart resolved")
	})

	t.Run("Enrich/consume rejects unknown method", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "CRDT",
				Methods: []generator.MethodInfo{{Name: "Merge"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Merge",
						Directives: []directive.Directive{
							{Name: directive.CRDTMerge, Args: []string{"Missing"}},
						},
					},
				},
			},
		}
		err := spec.Enrich(data, nil)
		testkit.Error(t, err, "unknown counterpart method")
	})
}
