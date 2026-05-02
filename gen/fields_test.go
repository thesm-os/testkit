// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen_test

import (
	"go/types"
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

func TestSampleBasicValue(t *testing.T) {
	t.Parallel()

	t.Run("string", func(t *testing.T) {
		t.Parallel()
		got := gen.SampleBasicValue(types.Typ[types.String], "Name")
		testkit.Equal(t, got, `"test-name"`, "string sample")
	})

	t.Run("bool", func(t *testing.T) {
		t.Parallel()
		got := gen.SampleBasicValue(types.Typ[types.Bool], "Active")
		testkit.Equal(t, got, "true", "bool sample")
	})

	t.Run("int", func(t *testing.T) {
		t.Parallel()
		got := gen.SampleBasicValue(types.Typ[types.Int], "Count")
		testkit.Equal(t, got, "42", "int sample")
	})

	t.Run("float64", func(t *testing.T) {
		t.Parallel()
		got := gen.SampleBasicValue(types.Typ[types.Float64], "Score")
		testkit.Equal(t, got, "3.14", "float sample")
	})

	t.Run("uint", func(t *testing.T) {
		t.Parallel()
		got := gen.SampleBasicValue(types.Typ[types.Uint], "Size")
		// uint has IsInteger set, so it returns 42
		testkit.Equal(t, got, "42", "uint sample")
	})
}

func TestSampleValueOf(t *testing.T) {
	t.Parallel()
	tracker := gen.NewImportTracker("example.com/test")

	t.Run("basic string", func(t *testing.T) {
		t.Parallel()
		got := gen.SampleValueOf(types.Typ[types.String], "Name", tracker)
		testkit.Equal(t, got, `"test-name"`, "string sample")
	})

	t.Run("basic int", func(t *testing.T) {
		t.Parallel()
		got := gen.SampleValueOf(types.Typ[types.Int], "Count", tracker)
		testkit.Equal(t, got, "42", "int sample")
	})

	t.Run("basic bool", func(t *testing.T) {
		t.Parallel()
		got := gen.SampleValueOf(types.Typ[types.Bool], "Active", tracker)
		testkit.Equal(t, got, "true", "bool sample")
	})

	t.Run("slice", func(t *testing.T) {
		t.Parallel()
		sl := types.NewSlice(types.Typ[types.String])
		got := gen.SampleValueOf(sl, "Names", tracker)
		testkit.Assert(t, got).Contains("[]string", "must have slice type")
	})

	t.Run("map", func(t *testing.T) {
		t.Parallel()
		mp := types.NewMap(types.Typ[types.String], types.Typ[types.Int])
		got := gen.SampleValueOf(mp, "Data", tracker)
		testkit.Assert(t, got).Contains("map[string]int", "must have map type")
	})

	t.Run("pointer is nil", func(t *testing.T) {
		t.Parallel()
		ptr := types.NewPointer(types.Typ[types.String])
		got := gen.SampleValueOf(ptr, "Ref", tracker)
		testkit.Equal(t, got, "nil", "pointer sample is nil")
	})

	t.Run("func is nil", func(t *testing.T) {
		t.Parallel()
		sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
		got := gen.SampleValueOf(sig, "Fn", tracker)
		testkit.Equal(t, got, "nil", "func sample is nil")
	})

	t.Run("chan is nil", func(t *testing.T) {
		t.Parallel()
		ch := types.NewChan(types.SendRecv, types.Typ[types.Int])
		got := gen.SampleValueOf(ch, "Ch", tracker)
		testkit.Equal(t, got, "nil", "chan sample is nil")
	})

	t.Run("interface is nil", func(t *testing.T) {
		t.Parallel()
		iface := types.NewInterfaceType(nil, nil)
		iface.Complete()
		got := gen.SampleValueOf(iface, "I", tracker)
		testkit.Equal(t, got, "nil", "interface sample is nil")
	})

	t.Run("named struct uses zero suffix", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		iface, err := pkg.Interface("Store")
		testkit.NoError(t, err, "must load Store")
		var getMethod gen.MethodInfo
		for _, m := range iface.Methods {
			if m.Name == "Get" {
				getMethod = m
			}
		}
		resultType := getMethod.Signature.Results().At(0).Type()
		tr := gen.NewImportTracker("example.com/other")
		got := gen.SampleValueOf(resultType, "Result", tr)
		testkit.Assert(t, got).Contains("{}", "named struct sample has zero suffix")
	})

	t.Run("array uses zero suffix", func(t *testing.T) {
		t.Parallel()
		arr := types.NewArray(types.Typ[types.Int], 3)
		got := gen.SampleValueOf(arr, "Arr", tracker)
		testkit.Assert(t, got).Contains("{1}", "array sample")
	})
}

func TestHasUnexportedFields(t *testing.T) {
	t.Parallel()

	t.Run("struct with unexported field", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		s, err := pkg.Struct("Item")
		testkit.NoError(t, err, "must load Item")
		testkit.True(t, gen.HasUnexportedFields(s.Type), "Item has unexported field")
	})

	t.Run("all-exported struct", func(t *testing.T) {
		t.Parallel()
		strct := types.NewStruct([]*types.Var{
			types.NewField(0, nil, "Name", types.Typ[types.String], false),
		}, nil)
		testkit.False(t, gen.HasUnexportedFields(strct), "all exported")
	})

	t.Run("non-struct returns false", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, gen.HasUnexportedFields(types.Typ[types.Int]), "int is not struct")
	})
}
