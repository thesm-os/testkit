// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package percentiles_test

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/percentiles"
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

func TestPercentiles(t *testing.T) {
	t.Parallel()

	t.Run("consume attaches multiple budgets sorted ascending", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Open").Directives = []directive.Directive{
			{Name: directive.Percentiles, Args: []string{"p99=100us", "p50=10us", "p95=50us"}},
		}
		testkit.NoError(t, spec.Enrich(data, pkg), "Enrich")
		got, ok := percentiles.Get(methodByName(data, "Open"))
		testkit.True(t, ok, "payload attached")
		testkit.Len(t, got.Budgets, 3, "three budgets")
		testkit.Equal(t, got.Budgets[0].Percentile, 50, "first budget is p50 (sorted)")
		testkit.Equal(t, got.Budgets[1].Percentile, 95, "second budget is p95")
		testkit.Equal(t, got.Budgets[2].Percentile, 99, "third budget is p99")
		testkit.Equal(t, got.Budgets[0].Max, 10*time.Microsecond, "p50 duration")
		testkit.Equal(t, got.Budgets[2].Raw, "p99=100us", "Raw preserved for stable rendering")
	})

	t.Run("rejects empty arg list", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Open").Directives = []directive.Directive{
			{Name: directive.Percentiles, Args: nil},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "no args rejected")
		testkit.Assert(t, err.Error()).Contains("at least one", "diagnostic")
	})

	t.Run("rejects malformed budget (no equals)", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Open").Directives = []directive.Directive{
			{Name: directive.Percentiles, Args: []string{"p99-100us"}},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "missing = rejected")
	})

	t.Run("rejects non-p prefix", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Open").Directives = []directive.Directive{
			{Name: directive.Percentiles, Args: []string{"99=100us"}},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "missing p prefix rejected")
	})

	t.Run("rejects out-of-range percentile", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Open").Directives = []directive.Directive{
			{Name: directive.Percentiles, Args: []string{"p150=100us"}},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "p150 rejected")
	})

	t.Run("rejects unparseable duration", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Open").Directives = []directive.Directive{
			{Name: directive.Percentiles, Args: []string{"p99=forever"}},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "non-duration rejected")
	})

	t.Run("rejects duplicate percentile", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Open").Directives = []directive.Directive{
			{Name: directive.Percentiles, Args: []string{"p99=100us", "p99=200us"}},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "duplicate p99 rejected")
		testkit.Assert(t, err.Error()).Contains("declared twice", "diagnostic")
	})

	t.Run("rejects zero / negative", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Open").Directives = []directive.Directive{
			{Name: directive.Percentiles, Args: []string{"p99=0s"}},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "zero rejected")
	})

	t.Run("Get returns zero+false on absent attachment", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		got, ok := percentiles.Get(&m)
		testkit.False(t, ok, "missing attachment")
		testkit.Len(t, got.Budgets, 0, "zero payload")
	})

	t.Run("Has reflects presence in Attachments", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, percentiles.Has(&m), "absent without attachment")
		spec.Set(
			&m.Attachments,
			directive.Percentiles,
			percentiles.Payload{Budgets: []percentiles.Budget{{Percentile: 99, Max: time.Microsecond}}},
		)
		testkit.True(t, percentiles.Has(&m), "present after Set")
	})
}
