// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package twophase_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all"
	"go.thesmos.sh/testkit/generator/spec/twophase"
)

func TestTwoPhase(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.TwoPhase)) > 0, "two-phase consumer registered")
	})

	t.Run("Has/Get reflect attached payload", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, twophase.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.TwoPhase, twophase.Payload{Commit: "Commit", Rollback: "Rollback"})
		got, ok := twophase.Get(&m)
		testkit.True(t, ok, "Get returns ok=true when present")
		testkit.Equal(t, got.Commit, "Commit", "commit round-trip")
		testkit.Equal(t, got.Rollback, "Rollback", "rollback round-trip")
	})

	t.Run("Enrich resolves both siblings", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "DB",
				Methods: []generator.MethodInfo{{Name: "Begin"}, {Name: "Commit"}, {Name: "Rollback"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Begin",
						Directives: []directive.Directive{
							{Name: directive.TwoPhase, Args: []string{"Commit", "Rollback"}},
						},
					},
				},
			},
		}
		testkit.NoError(t, spec.Enrich(data, nil), "enrich succeeds")
		got, _ := twophase.Get(&data.Methods[0])
		testkit.Equal(t, got.Commit, "Commit", "commit resolved")
		testkit.Equal(t, got.Rollback, "Rollback", "rollback resolved")
	})

	t.Run("Enrich rejects missing commit", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "DB",
				Methods: []generator.MethodInfo{{Name: "Begin"}, {Name: "Rollback"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Begin",
						Directives: []directive.Directive{
							{Name: directive.TwoPhase, Args: []string{"NoCommit", "Rollback"}},
						},
					},
				},
			},
		}
		testkit.Error(t, spec.Enrich(data, nil), "missing commit")
	})

	t.Run("Enrich rejects missing rollback", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "DB",
				Methods: []generator.MethodInfo{{Name: "Begin"}, {Name: "Commit"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Begin",
						Directives: []directive.Directive{
							{Name: directive.TwoPhase, Args: []string{"Commit", "NoRollback"}},
						},
					},
				},
			},
		}
		testkit.Error(t, spec.Enrich(data, nil), "missing rollback")
	})

	t.Run("Enrich rejects wrong arg count", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Interface: generator.InterfaceInfo{
				Name:    "DB",
				Methods: []generator.MethodInfo{{Name: "Begin"}, {Name: "Commit"}},
			},
			Methods: []spec.Method{
				{
					MethodInfo: generator.MethodInfo{
						Name: "Begin",
						Directives: []directive.Directive{
							{Name: directive.TwoPhase, Args: []string{"Commit"}},
						},
					},
				},
			},
		}
		testkit.Error(t, spec.Enrich(data, nil), "wrong arg count")
	})
}
