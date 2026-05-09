// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package spec_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
)

func TestEnrich(t *testing.T) {
	t.Parallel()

	t.Run("dispatches registered consumers per directive", func(t *testing.T) {
		t.Parallel()
		// Use a fresh probe directive so we don't interact with the
		// shared sample registration.
		const probe = "spec-test-probe"
		var fired []string
		spec.RegisterConsumer(probe, func(
			m *spec.Method, _ directive.Directive,
			_ *spec.Data, _ *generator.Package,
		) error {
			fired = append(fired, m.Name)
			return nil
		})
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
		testkit.Equal(t, fired, []string{"Probe"}, "consumer fired exactly once")
	})

	t.Run("propagates consumer errors with method position context", func(t *testing.T) {
		t.Parallel()
		const probe = "spec-test-probe-err"
		want := errors.New("boom")
		spec.RegisterConsumer(probe, func(
			_ *spec.Method, _ directive.Directive,
			_ *spec.Data, _ *generator.Package,
		) error {
			return want
		})
		data := &spec.Data{
			Interface: generator.InterfaceInfo{Name: "Probe"},
			Methods: []spec.Method{{
				MethodInfo: generator.MethodInfo{
					Name:       "Probe",
					Directives: []directive.Directive{{Name: probe}},
				},
			}},
		}
		err := spec.Enrich(data, nil)
		testkit.True(t, err != nil, "non-nil error")
		testkit.True(t, errors.Is(err, want), "wraps the underlying error")
		testkit.Assert(t, err.Error()).
			Contains("Probe.Probe", "diagnostic names interface.method")
	})

	t.Run("Consumers returns the registered slice (copy)", func(t *testing.T) {
		t.Parallel()
		const probe = "spec-test-probe-list"
		spec.RegisterConsumer(probe, func(
			_ *spec.Method, _ directive.Directive,
			_ *spec.Data, _ *generator.Package,
		) error {
			return nil
		})
		got := spec.Consumers(probe)
		testkit.Len(t, got, 1, "one consumer registered")
	})
}
