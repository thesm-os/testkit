// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package marker_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/marker"
)

func TestRegister(t *testing.T) {
	t.Parallel()

	// Use a probe directive name so tests don't depend on the
	// shared registration timing of other markers.
	const probe = "marker-test-probe"
	marker.Register(probe)

	t.Run("attaches Presence on directive match", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Methods: []spec.Method{{
				MethodInfo: generator.MethodInfo{
					Name:       "Probe",
					Directives: []directive.Directive{{Name: probe}},
				},
			}},
		}
		err := spec.Enrich(data, nil)
		testkit.NoError(t, err, "Enrich")
		_, ok := spec.Get[marker.Presence](data.Methods[0].Attachments, probe)
		testkit.True(t, ok, "Presence attached")
		testkit.True(t, spec.Has(data.Methods[0].Attachments, probe), "Has confirms presence")
	})

	t.Run("attaches nothing when directive absent", func(t *testing.T) {
		t.Parallel()
		data := &spec.Data{
			Methods: []spec.Method{{
				MethodInfo: generator.MethodInfo{Name: "Quiet"},
			}},
		}
		err := spec.Enrich(data, nil)
		testkit.NoError(t, err, "Enrich")
		testkit.False(t, spec.Has(data.Methods[0].Attachments, probe),
			"no presence without the directive")
	})
}
