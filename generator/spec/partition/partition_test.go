// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package partition_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/partition"
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

	t.Run("resolves direct param name (param == directive arg)", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		testkit.NoError(t, spec.Enrich(data, pkg), "Enrich")
		got, ok := spec.Get[partition.Payload](
			methodByName(data, "ShardByKey").Attachments, directive.Partition)
		testkit.True(t, ok, "payload attached")
		testkit.Equal(t, got.FieldPath, "key", "direct param name")
		testkit.Equal(t, got.FieldName, "key", "field name")
		testkit.Equal(t, got.FieldType, "string", "type")
	})

	t.Run("resolves struct field on a struct-typed param", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		testkit.NoError(t, spec.Enrich(data, pkg), "Enrich")
		got, ok := spec.Get[partition.Payload](
			methodByName(data, "Shard").Attachments, directive.Partition)
		testkit.True(t, ok, "payload attached")
		testkit.Equal(t, got.FieldPath, "item.ID", "param.Field path")
		testkit.Equal(t, got.FieldName, "ID", "field name")
		testkit.Equal(t, got.FieldType, "string", "field type")
	})

	t.Run("rejects unknown field", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Shard").Directives = []directive.Directive{
			{Name: directive.Partition, Args: []string{"DoesNotExist"}},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "unknown field rejected")
		testkit.Assert(t, err.Error()).Contains("not found in parameters", "diagnostic")
	})

	t.Run("anonymous params: ParamName fallback synthesizes pN", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		testkit.NoError(t, spec.Enrich(data, pkg), "Enrich")
		got, ok := spec.Get[partition.Payload](
			methodByName(data, "ShardAnon").Attachments, directive.Partition)
		testkit.True(t, ok, "payload attached")
		// First non-ctx param (p1, string) is skipped; struct param
		// p2 carries the field. Path uses synthesized pN names.
		testkit.Equal(t, got.FieldPath, "p2.ID", "ParamName(2)+field")
	})

	t.Run("variadic method: trailing variadic excluded from walk", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		testkit.NoError(t, spec.Enrich(data, pkg), "Enrich")
		got, ok := spec.Get[partition.Payload](
			methodByName(data, "Batch").Attachments, directive.Partition)
		testkit.True(t, ok, "payload attached")
		testkit.Equal(t, got.FieldPath, "tenant", "resolved against non-variadic param")
	})

	t.Run("rejects wrong arg count", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadWorkflow(t)
		methodByName(data, "Shard").Directives = []directive.Directive{
			{Name: directive.Partition, Args: nil},
		}
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "no args rejected")
	})
}
