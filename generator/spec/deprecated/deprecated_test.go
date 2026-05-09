// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package deprecated_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/deprecated"
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

func TestDeprecated(t *testing.T) {
	t.Parallel()

	t.Run("consume attaches the replacement payload", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		testkit.NoError(t, spec.Enrich(data, pkg), "Enrich")
		got, ok := deprecated.Get(methodByName(data, "Legacy"))
		testkit.True(t, ok, "deprecated payload attached")
		testkit.Equal(t, got.Replacement, "Submit", "replacement name")
	})

	t.Run("consume rejects arg count != 1", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		m := methodByName(data, "Legacy")
		m.Directives = []directive.Directive{
			{Name: directive.Deprecated, Args: []string{"A", "B"}},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "two args rejected")
		testkit.Assert(t, err.Error()).Contains("expects 1 arg", "diagnostic")
	})

	t.Run("Get returns zero+false on absent attachment", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		got, ok := deprecated.Get(&m)
		testkit.False(t, ok, "missing attachment")
		testkit.Equal(t, got.Replacement, "", "zero payload")
	})

	t.Run("Has reflects presence in Attachments", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, deprecated.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.Deprecated, deprecated.Payload{Replacement: "X"})
		testkit.True(t, deprecated.Has(&m), "present after Set")
	})
}
