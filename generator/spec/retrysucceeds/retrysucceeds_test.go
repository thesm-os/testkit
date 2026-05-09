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

func TestRetrySucceeds(t *testing.T) {
	t.Parallel()

	t.Run("consume attaches the parsed attempt count", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		testkit.NoError(t, spec.Enrich(data, pkg), "Enrich")
		got, ok := retrysucceeds.Get(methodByName(data, "Retry"))
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

	t.Run("Get returns zero+false on absent attachment", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		got, ok := retrysucceeds.Get(&m)
		testkit.False(t, ok, "missing attachment")
		testkit.Equal(t, got.N, 0, "zero payload")
	})

	t.Run("Has reflects presence in Attachments", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, retrysucceeds.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.RetrySucceedsOnAttempt, retrysucceeds.Payload{N: 2})
		testkit.True(t, retrysucceeds.Has(&m), "present after Set")
	})
}
