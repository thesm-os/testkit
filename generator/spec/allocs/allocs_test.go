// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package allocs_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/allocs"
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

func TestAllocs(t *testing.T) {
	t.Parallel()

	t.Run("consume attaches the parsed budget", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Open").Directives = []directive.Directive{
			{Name: directive.Allocs, Args: []string{"3"}},
		}
		testkit.NoError(t, spec.Enrich(data, pkg), "Enrich")
		got, ok := allocs.Get(methodByName(data, "Open"))
		testkit.True(t, ok, "payload attached")
		testkit.Equal(t, got.Max, 3, "Max parsed")
	})

	t.Run("zero is allowed", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Open").Directives = []directive.Directive{
			{Name: directive.Allocs, Args: []string{"0"}},
		}
		testkit.NoError(t, spec.Enrich(data, pkg), "Enrich")
		got, ok := allocs.Get(methodByName(data, "Open"))
		testkit.True(t, ok, "payload attached")
		testkit.Equal(t, got.Max, 0, "zero is a valid budget — alloc-free assertion")
	})

	t.Run("rejects non-integer arg", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Open").Directives = []directive.Directive{
			{Name: directive.Allocs, Args: []string{"three"}},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "non-integer rejected")
		testkit.Assert(t, err.Error()).Contains("not a non-negative integer", "diagnostic")
	})

	t.Run("rejects negative", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Open").Directives = []directive.Directive{
			{Name: directive.Allocs, Args: []string{"-1"}},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "negative rejected")
	})

	t.Run("rejects wrong arg count", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Open").Directives = []directive.Directive{
			{Name: directive.Allocs, Args: nil},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "no args rejected")
	})

	t.Run("Get returns zero+false on absent attachment", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		got, ok := allocs.Get(&m)
		testkit.False(t, ok, "missing attachment")
		testkit.Equal(t, got.Max, 0, "zero payload")
	})

	t.Run("Has reflects presence in Attachments", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, allocs.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.Allocs, allocs.Payload{Max: 5})
		testkit.True(t, allocs.Has(&m), "present after Set")
	})
}
