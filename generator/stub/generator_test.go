// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub_test

import (
	"slices"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/stub"
)

func loadFixture(t *testing.T, name string) *generator.Package {
	t.Helper()
	pkg, err := generator.NewLoader().Load("./../testdata/"+name, "")
	testkit.NoError(t, err, "Load testdata/"+name)
	return pkg
}

func TestGenerator(t *testing.T) {
	t.Parallel()

	t.Run("Name returns subcommand", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, (&stub.Generator{}).Name(), stub.Name, "Name")
	})

	t.Run("Generate emits two files: impl + auto-test", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&stub.Generator{}).Generate(pkg, []string{"Store"},
			generator.DefaultConfig(), generator.Options{
				Output: "storetest/store.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		testkit.Len(t, res.Files, 2, "impl + test")

		paths := make([]string, len(res.Files))
		for i, f := range res.Files {
			paths[i] = f.Path
		}
		testkit.Equal(t, paths[0], "storetest/store.gen.go", "impl path")
		testkit.True(t, slices.Contains(paths, "storetest/store.gen_test.go"),
			"test path is auto-derived via TestPathFrom")
	})

	t.Run("impl emits StubX type, per-method stub, dispatch, fault helpers", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&stub.Generator{}).Generate(pkg, []string{"Store"},
			generator.DefaultConfig(), generator.Options{
				Output: "storetest/store.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		impl := string(res.Files[0].Content)
		testkit.Assert(t, impl).
			Contains("type StoreStub struct", "stub type emitted").
			Contains("*StoreStubGet", "per-method stub field for Get").
			Contains("*StoreStubPut", "per-method stub field for Put").
			Contains("type StoreStubGetCall struct", "Get's Call struct (stub-prefixed for collision-safety)").
			Contains("type StoreStubPutCall struct", "Put's Call struct (stub-prefixed)").
			Contains("type StoreStubGet struct", "per-method stub type").
			Contains("*stub.MethodStub[StoreStubGetCall]",
				"per-method stub embeds runtime MethodStub").
			Contains("func NewStoreStub", "constructor").
			Contains("func StoreStubDelegateTo", "DelegateTo option").
			Contains("func WithStoreGet", "constructor option per method").
			Contains("func (s *StoreStub) Get(ctx context.Context, key string) (basic.Item, error)",
				"Get implements interface").
			Contains("s.OnGet.Record(call)", "Record on dispatch").
			Contains("s.OnGet.FailUnexpectedCall(call)", "fail-unexpected path").
			Contains("func (s *StoreStubGet) FaultNotFound()",
				"sentinel fault helper from //testkit:errors")
	})

	t.Run("test file lives in sibling _test pkg", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&stub.Generator{}).Generate(pkg, []string{"Store"},
			generator.DefaultConfig(), generator.Options{
				Output: "storetest/store.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		var test string
		for _, f := range res.Files {
			if strings.HasSuffix(f.Path, "_test.go") {
				test = string(f.Content)
			}
		}
		testkit.Assert(t, test).
			Contains("package storetest_test", "external _test package").
			Contains("storetest.NewStoreStub", "calls go through GenQualifier")
	})

	t.Run("non-interface arg surfaces a hard error", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		_, err := (&stub.Generator{}).Generate(pkg, []string{"Status"},
			generator.DefaultConfig(), generator.Options{
				Output: "x.gen.go",
			})
		testkit.True(t, err != nil, "non-interface rejected")
	})
}
