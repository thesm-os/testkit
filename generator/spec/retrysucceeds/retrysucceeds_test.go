// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package retrysucceeds_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/retrysucceeds"
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

	t.Run("happy path: positive integer attached", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		testkit.NoError(t, spec.Enrich(data, pkg), "Enrich")
		got, ok := spec.Get[retrysucceeds.Payload](
			methodByName(data, "Retry").Attachments, directive.RetrySucceedsOnAttempt)
		testkit.True(t, ok, "payload attached")
		testkit.Equal(t, got.N, 3, "N parsed")
	})

	t.Run("rejects non-integer arg", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Retry").Directives = []directive.Directive{
			{Name: directive.RetrySucceedsOnAttempt, Args: []string{"three"}},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "non-integer rejected")
		testkit.Assert(t, err.Error()).Contains("not a positive integer", "diagnostic")
	})

	t.Run("rejects zero / negative", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Retry").Directives = []directive.Directive{
			{Name: directive.RetrySucceedsOnAttempt, Args: []string{"0"}},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "zero rejected")
	})

	t.Run("rejects wrong arg count", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Retry").Directives = []directive.Directive{
			{Name: directive.RetrySucceedsOnAttempt, Args: nil},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "no args rejected")
	})
}
