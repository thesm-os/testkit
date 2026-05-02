// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
)

func TestBuildParamFields(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "basic")
	iface, err := pkg.Interface("Store")
	testkit.NoError(t, err, "must load Store")

	// Get(ctx context.Context, id string) (Item, error)
	var getMethod gen.MethodInfo
	for _, m := range iface.Methods {
		if m.Name == "Get" {
			getMethod = m
			break
		}
	}

	t.Run("builds fields from Get params", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/test")
		fields := gen.BuildParamFields(getMethod.Signature.Params(), tracker)
		testkit.Len(t, fields, 2, "Get has 2 params")
		testkit.Equal(t, fields[0].FieldName, "Ctx", "first param is ctx → Ctx")
		testkit.Equal(t, fields[1].FieldName, "ID", "second param is id → ID (initialism)")
	})

	t.Run("fields have type strings", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/test")
		fields := gen.BuildParamFields(getMethod.Signature.Params(), tracker)
		testkit.Assert(t, fields[0].TypeStr).Contains("Context", "must have Context type")
		testkit.Equal(t, fields[1].TypeStr, "string", "id is string")
	})
}

func TestBuildResultFields(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "basic")
	iface, err := pkg.Interface("Store")
	testkit.NoError(t, err, "must load Store")

	var getMethod, putMethod gen.MethodInfo
	for _, m := range iface.Methods {
		switch m.Name {
		case "Get":
			getMethod = m
		case "Put":
			putMethod = m
		}
	}

	t.Run("Get has Result and Err fields", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/test")
		fields := gen.BuildResultFields(getMethod.Signature.Results(), tracker)
		testkit.Len(t, fields, 2, "Get returns (Item, error)")
		testkit.Equal(t, fields[0].FieldName, "Result", "first non-error result is Result")
		testkit.True(t, fields[0].ZeroValue != "", "must have zero value")
		testkit.Equal(t, fields[1].FieldName, "Err", "error return is Err")
		testkit.True(t, fields[1].IsError, "must flag error field")
	})

	t.Run("Put has only Err field", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/test")
		fields := gen.BuildResultFields(putMethod.Signature.Results(), tracker)
		testkit.Len(t, fields, 1, "Put returns error")
		testkit.Equal(t, fields[0].FieldName, "Err", "single error return")
		testkit.True(t, fields[0].IsError, "must flag error")
	})

	t.Run("non-error field has zero value", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/test")
		fields := gen.BuildResultFields(getMethod.Signature.Results(), tracker)
		testkit.False(t, fields[0].IsError, "Result is not error")
		testkit.Assert(t, fields[0].ZeroValue).IsNotEmpty("must have zero value")
	})
}
