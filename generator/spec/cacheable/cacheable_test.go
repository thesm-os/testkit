// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cacheable_test

import (
	"go/types"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all"
	"go.thesmos.sh/testkit/generator/spec/cacheable"
)

func TestCacheable(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.Cacheable)) > 0,
			"cacheable consumer registered")
	})

	t.Run("Has reflects presence in Attachments", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, cacheable.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.Cacheable, struct{}{})
		testkit.True(t, cacheable.Has(&m), "present after Set")
	})

	t.Run("consume attaches when method has non-error result", func(t *testing.T) {
		t.Parallel()
		sig := types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int])),
			false)
		data := &spec.Data{
			Interface: generator.InterfaceInfo{Name: "Iface"},
			Methods: []spec.Method{{
				MethodInfo: generator.MethodInfo{
					Name:       "Score",
					Signature:  sig,
					Directives: []directive.Directive{{Name: directive.Cacheable}},
				},
			}},
		}
		testkit.NoError(t, spec.Enrich(data, nil), "Enrich")
		testkit.True(t, cacheable.Has(&data.Methods[0]), "cacheable attached")
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
					Directives: []directive.Directive{{Name: directive.Cacheable}},
				},
			}},
		}
		testkit.True(t, spec.Enrich(data, nil) != nil, "void method rejected")
	})
}
