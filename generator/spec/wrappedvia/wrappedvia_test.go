// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package wrappedvia_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/wrappedvia"
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

func TestConsume(t *testing.T) {
	t.Parallel()

	t.Run("happy path: wrap target attached", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		testkit.NoError(t, spec.Enrich(data, pkg), "Enrich")
		got, ok := spec.Get[wrappedvia.Payload](
			methodByName(data, "Wrap").Attachments, directive.WrappedVia)
		testkit.True(t, ok, "payload attached")
		testkit.Equal(t, got.VarName, "ErrForbidden", "VarName")
		testkit.Equal(t, got.Qualified, "ErrForbidden", "local: bare")
	})

	t.Run("rejects wrong arg count", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Wrap").Directives = []directive.Directive{
			{Name: directive.WrappedVia, Args: []string{"A", "B"}},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "two args rejected")
	})

	t.Run("rejects unknown sentinel", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Wrap").Directives = []directive.Directive{
			{Name: directive.WrappedVia, Args: []string{"DoesNotExist"}},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "unknown rejected")
	})

	t.Run("rejects non-error symbol (resolves but VarOfType fails)", func(t *testing.T) {
		t.Parallel()
		// Item is a struct type, not a var. Resolver finds it; the
		// VarOfType validator rejects it.
		pkg, data := loadWorkflow(t)
		methodByName(data, "Wrap").Directives = []directive.Directive{
			{Name: directive.WrappedVia, Args: []string{"Item"}},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "non-var rejected")
		testkit.Assert(t, err.Error()).Contains("not a variable", "diagnostic")
	})
}
