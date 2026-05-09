// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package monotonic_test

import (
	"go/types"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all"
	"go.thesmos.sh/testkit/generator/spec/monotonic"
)

func TestMonotonic(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.Monotonic)) > 0,
			"monotonic consumer registered")
	})

	t.Run("Has reflects presence in Attachments", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, monotonic.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.Monotonic, struct{}{})
		testkit.True(t, monotonic.Has(&m), "present after Set")
	})

	t.Run("consume attaches when result is ordered", func(t *testing.T) {
		t.Parallel()
		// Count() (int, error) — int is ordered.
		sig := types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(
				types.NewVar(0, nil, "", types.Typ[types.Int]),
				types.NewVar(0, nil, "", types.Universe.Lookup("error").Type()),
			), false)
		data := &spec.Data{
			Interface: generator.InterfaceInfo{Name: "Iface"},
			Methods: []spec.Method{{
				MethodInfo: generator.MethodInfo{
					Name:       "Count",
					Signature:  sig,
					Directives: []directive.Directive{{Name: directive.Monotonic}},
				},
			}},
		}
		testkit.NoError(t, spec.Enrich(data, nil), "Enrich")
		testkit.True(t, monotonic.Has(&data.Methods[0]), "monotonic attached")
	})

	t.Run("consume rejects non-ordered result", func(t *testing.T) {
		t.Parallel()
		// Info() (bool, error) — bool is not ordered.
		sig := types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(
				types.NewVar(0, nil, "", types.Typ[types.Bool]),
				types.NewVar(0, nil, "", types.Universe.Lookup("error").Type()),
			), false)
		data := &spec.Data{
			Interface: generator.InterfaceInfo{Name: "Iface"},
			Methods: []spec.Method{{
				MethodInfo: generator.MethodInfo{
					Name:       "Info",
					Signature:  sig,
					Directives: []directive.Directive{{Name: directive.Monotonic}},
				},
			}},
		}
		testkit.True(t, spec.Enrich(data, nil) != nil,
			"non-ordered result type rejected")
	})

	t.Run("consume rejects void method", func(t *testing.T) {
		t.Parallel()
		sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
		data := &spec.Data{
			Interface: generator.InterfaceInfo{Name: "Iface"},
			Methods: []spec.Method{{
				MethodInfo: generator.MethodInfo{
					Name:       "Reset",
					Signature:  sig,
					Directives: []directive.Directive{{Name: directive.Monotonic}},
				},
			}},
		}
		testkit.True(t, spec.Enrich(data, nil) != nil,
			"void method rejected")
	})
}
