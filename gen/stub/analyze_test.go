// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
	"go.thesmos.sh/testkit/gen/stub"
)

func analyzeBasic(t *testing.T) *stub.Data {
	t.Helper()
	pkg := loadTestPackage(t, "basic")
	data, err := stub.Analyze(pkg, []string{"Store"}, gen.DefaultConfig(), gen.Options{
		Output: "storetest/store_stub.gen.go",
	})
	testkit.NoError(t, err, "must analyze")
	return data
}

func TestAnalyze(t *testing.T) {
	t.Parallel()

	t.Run("produces correct interface data", func(t *testing.T) {
		t.Parallel()
		data := analyzeBasic(t)
		testkit.Len(t, data.Interfaces, 1, "must have 1 interface")
		testkit.Equal(t, data.Interfaces[0].Name, "Store", "interface name")
		testkit.Equal(t, data.Interfaces[0].StubName, "StoreStub", "stub name")
	})

	t.Run("method naming conventions", func(t *testing.T) {
		t.Parallel()
		data := analyzeBasic(t)
		methods := data.Interfaces[0].Methods
		testkit.Len(t, methods, 3, "Store has 3 methods")

		testkit.Equal(t, methods[0].CallType, "StoreDeleteCall", "Delete call type")
		testkit.Equal(t, methods[0].StubType, "StoreDeleteStub", "Delete stub type")

		testkit.Equal(t, methods[1].CallType, "StoreGetCall", "Get call type")
		testkit.Equal(t, methods[1].StubType, "StoreGetStub", "Get stub type")

		testkit.Equal(t, methods[2].CallType, "StorePutCall", "Put call type")
		testkit.Equal(t, methods[2].StubType, "StorePutStub", "Put stub type")
	})

	t.Run("params and results populated", func(t *testing.T) {
		t.Parallel()
		data := analyzeBasic(t)
		var get *stub.MethodData
		for _, m := range data.Interfaces[0].Methods {
			if m.Name == "Get" {
				get = m
			}
		}
		testkit.Len(t, get.Params, 2, "Get has 2 params")
		testkit.Equal(t, get.Params[0].FieldName, "Ctx", "first param field")
		testkit.Equal(t, get.Params[1].FieldName, "ID", "second param field")
		testkit.Len(t, get.Results, 2, "Get has 2 results")
		testkit.Equal(t, get.Results[0].FieldName, "Result", "first result field")
		testkit.Equal(t, get.Results[1].FieldName, "Err", "second result field")
	})

	t.Run("package name from output path", func(t *testing.T) {
		t.Parallel()
		data := analyzeBasic(t)
		testkit.Equal(t, data.PackageName, "storetest", "package name from subdir")
	})

	t.Run("imports include testing and testkit", func(t *testing.T) {
		t.Parallel()
		data := analyzeBasic(t)
		paths := make([]string, len(data.Imports))
		for i, imp := range data.Imports {
			paths[i] = imp.Path
		}
		testkit.Assert(t, paths).Contains("testing", "must import testing")
		testkit.Assert(t, paths).Contains("go.thesmos.sh/testkit", "must import testkit")
	})

	t.Run("directives attached to methods", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "directives")
		data, err := stub.Analyze(pkg, []string{"Store"}, gen.DefaultConfig(), gen.Options{
			Output: "storetest/store_stub.gen.go",
		})
		testkit.NoError(t, err, "must analyze")

		var get *stub.MethodData
		for _, m := range data.Interfaces[0].Methods {
			if m.Name == "Get" {
				get = m
			}
		}
		testkit.Len(t, get.Directives, 1, "Get has 1 directive")
		testkit.Equal(t, get.Directives[0].Name, "errors", "directive name")
		testkit.Equal(t, get.Directives[0].Args, []string{"ErrNotFound", "ErrConflict"}, "directive args")
	})

	t.Run("return type is unexported", func(t *testing.T) {
		t.Parallel()
		data := analyzeBasic(t)
		var get *stub.MethodData
		for _, m := range data.Interfaces[0].Methods {
			if m.Name == "Get" {
				get = m
			}
		}
		testkit.Equal(t, get.ReturnType, "storeGetReturn", "return type must be unexported")
	})

	t.Run("qualified type includes package", func(t *testing.T) {
		t.Parallel()
		data := analyzeBasic(t)
		testkit.Assert(t, data.Interfaces[0].QualifiedType).
			Contains("Store", "must contain type name")
	})
}
