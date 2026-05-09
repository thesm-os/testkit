// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package latency_test

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/latency"
)

func loadWorkflow(t *testing.T) (*generator.Package, *spec.Data) {
	t.Helper()
	pkg, err := generator.NewLoader().Load("./../../testdata/basic", "")
	testkit.NoError(t, err, "Load testdata/basic")
	data, err := spec.Analyze(pkg, []string{"Workflow"},
		generator.DefaultConfig(), generator.Options{Output: "workflowtest/x.gen.go"})
	testkit.NoError(t, err, "Analyze")
	return pkg, data
}

func methodByName(data *spec.Data, name string) *spec.Method {
	for i := range data.Methods {
		if data.Methods[i].Name == name {
			return &data.Methods[i]
		}
	}
	return nil
}

func TestLatency(t *testing.T) {
	t.Parallel()

	t.Run("consume attaches the parsed duration", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Open").Directives = []directive.Directive{
			{Name: directive.Latency, Args: []string{"100us"}},
		}
		testkit.NoError(t, spec.Enrich(data, pkg), "Enrich")
		got, ok := latency.Get(methodByName(data, "Open"))
		testkit.True(t, ok, "payload attached")
		testkit.Equal(t, got.Max, 100*time.Microsecond, "Max parsed")
		testkit.Equal(t, got.Raw, "100us", "Raw preserved for stable rendering")
	})

	t.Run("rejects unparseable duration", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Open").Directives = []directive.Directive{
			{Name: directive.Latency, Args: []string{"forever"}},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "non-duration rejected")
	})

	t.Run("rejects zero", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Open").Directives = []directive.Directive{
			{Name: directive.Latency, Args: []string{"0s"}},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "zero rejected")
		testkit.Assert(t, err.Error()).Contains("must be positive", "diagnostic")
	})

	t.Run("rejects negative", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Open").Directives = []directive.Directive{
			{Name: directive.Latency, Args: []string{"-1ms"}},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "negative rejected")
	})

	t.Run("rejects wrong arg count", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Open").Directives = []directive.Directive{
			{Name: directive.Latency, Args: nil},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "no args rejected")
	})

	t.Run("Get returns zero+false on absent attachment", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		got, ok := latency.Get(&m)
		testkit.False(t, ok, "missing attachment")
		testkit.Equal(t, got.Max, time.Duration(0), "zero payload")
		testkit.Equal(t, got.Raw, "", "zero raw")
	})

	t.Run("Has reflects presence in Attachments", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, latency.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.Latency, latency.Payload{Max: time.Millisecond, Raw: "1ms"})
		testkit.True(t, latency.Has(&m), "present after Set")
	})
}
