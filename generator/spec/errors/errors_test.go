// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package errors_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/errors"
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

	t.Run("happy path: sentinels resolved with bare and short names", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		testkit.NoError(t, spec.Enrich(data, pkg), "Enrich")
		got, ok := spec.Get[errors.Payload](
			methodByName(data, "Submit").Attachments, directive.Errors)
		testkit.True(t, ok, "payload attached")
		testkit.Len(t, got.Sentinels, 2, "two sentinels resolved")
		testkit.Equal(t, got.Sentinels[0].VarName, "ErrNotFound", "VarName")
		testkit.Equal(t, got.Sentinels[0].ShortName, "NotFound", "ShortName strips Err")
		testkit.Equal(t, got.Sentinels[0].Qualified, "ErrNotFound", "local: bare")
		testkit.Equal(t, got.Sentinels[1].VarName, "ErrConflict", "second VarName")
	})

	t.Run("rejects empty arg list", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Submit").Directives = []directive.Directive{
			{Name: directive.Errors, Args: nil},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "no args rejected")
		testkit.Assert(t, err.Error()).Contains("at least one sentinel", "diagnostic")
	})

	t.Run("rejects unknown sentinel", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Submit").Directives = []directive.Directive{
			{Name: directive.Errors, Args: []string{"DoesNotExist"}},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "unknown sentinel rejected")
	})

	t.Run("rejects ShortName collision", func(t *testing.T) {
		t.Parallel()
		// Two distinct sentinels both producing FaultNotFound is a
		// hard error. Build a synthetic case: ErrNotFound twice.
		pkg, data := loadWorkflow(t)
		methodByName(data, "Submit").Directives = []directive.Directive{
			{Name: directive.Errors, Args: []string{"ErrNotFound", "ErrNotFound"}},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "collision rejected")
		testkit.Assert(t, err.Error()).Contains("FaultNotFound", "diagnostic")
	})

	t.Run("rejects non-error variable", func(t *testing.T) {
		t.Parallel()
		// SentinelEOF is io.EOF (assignable to error) — this should
		// PASS. Use Item (a struct type) to fail. Item isn't a var
		// though — it's a type. Resolver returns a type object, then
		// VarOfType rejects.
		pkg, data := loadWorkflow(t)
		methodByName(data, "Submit").Directives = []directive.Directive{
			{Name: directive.Errors, Args: []string{"Item"}},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "non-var rejected")
	})
}
