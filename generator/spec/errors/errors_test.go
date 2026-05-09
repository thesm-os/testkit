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

func TestErrors(t *testing.T) {
	t.Parallel()

	t.Run("consume resolves sentinels with bare and short names", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		testkit.NoError(t, spec.Enrich(data, pkg), "Enrich")
		got, ok := errors.Get(methodByName(data, "Submit"))
		testkit.True(t, ok, "payload attached")
		testkit.Len(t, got.Sentinels, 2, "two sentinels resolved")
		testkit.Equal(t, got.Sentinels[0].VarName, "ErrNotFound", "VarName")
		testkit.Equal(t, got.Sentinels[0].ShortName, "NotFound", "ShortName strips Err")
		// Output is in workflowtest/, source is basic/ — local
		// refs qualify with the source-pkg alias.
		testkit.Equal(t, got.Sentinels[0].Qualified, "basic.ErrNotFound",
			"source-qualified for sibling output pkg")
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

	t.Run("Get returns zero+false on absent attachment", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		got, ok := errors.Get(&m)
		testkit.False(t, ok, "missing attachment")
		testkit.Len(t, got.Sentinels, 0, "zero payload")
	})

	t.Run("Has reflects presence in Attachments", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, errors.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.Errors, errors.Payload{})
		testkit.True(t, errors.Has(&m), "present after Set")
	})

	t.Run("FaultReturn delegates to Method.FaultReturn with sentinel expr", func(t *testing.T) {
		t.Parallel()
		// Submit(ctx, Item) error — error-only result, so FaultReturn
		// renders just the sentinel's qualified expression.
		pkg, data := loadWorkflow(t)
		testkit.NoError(t, spec.Enrich(data, pkg), "Enrich")
		submit := methodByName(data, "Submit")
		payload, _ := errors.Get(submit)
		got := errors.FaultReturn(submit, data.Tracker, payload.Sentinels[0])
		testkit.Equal(t, got, "basic.ErrNotFound",
			"error-only method renders sentinel expr alone")
	})
}
