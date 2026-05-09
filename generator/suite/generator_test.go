// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/suite"
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
		testkit.Equal(t, (&suite.Generator{}).Name(), suite.Name, "Name")
	})

	t.Run("emits one file at the configured Output path", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&suite.Generator{}).Generate(pkg, []string{"Store"},
			generator.DefaultConfig(), generator.Options{
				Output: "storetest/store_spec.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		testkit.Len(t, res.Files, 1, "single output file")
		testkit.Equal(t, res.Files[0].Path, "storetest/store_spec.gen.go", "configured path")
	})

	t.Run("driver functions follow Assert<Iface>Contract naming", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&suite.Generator{}).Generate(pkg, []string{"Store"},
			generator.DefaultConfig(), generator.Options{
				Output: "storetest/store_spec.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		out := string(res.Files[0].Content)
		testkit.Assert(t, out).
			Contains("func AssertStoreContract(t *testing.T, factory StoreFactory)",
				"single-impl driver").
			Contains("func AssertStoreContractAcrossImpls(t *testing.T, factories ...StoreNamedFactory)",
				"multi-impl driver").
			Contains("type StoreFactory func() basic.Store", "factory alias").
			Contains("type StoreNamedFactory struct", "named factory tuple")
	})

	t.Run("smoke subtests emitted per non-skip method", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&suite.Generator{}).Generate(pkg, []string{"Store"},
			generator.DefaultConfig(), generator.Options{
				Output: "storetest/store_spec.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		out := string(res.Files[0].Content)
		testkit.Assert(t, out).
			Contains(`t.Run("Get/smoke"`, "Get/smoke subtest").
			Contains(`t.Run("Put/smoke"`, "Put/smoke subtest")
	})

	t.Run("integration-only methods skipped", func(t *testing.T) {
		t.Parallel()
		pkg, err := generator.NewLoader().Load("./../testdata/interfaces", "")
		testkit.NoError(t, err, "Load interfaces fixture")
		res, err := (&suite.Generator{}).Generate(pkg, []string{"Directives"},
			generator.DefaultConfig(), generator.Options{
				Output: "directivestest/directives_spec.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		out := string(res.Files[0].Content)
		testkit.False(t, strings.Contains(out, `t.Run("Close/smoke"`),
			"Close is //testkit:integration-only — must be skipped")
	})

	t.Run("non-interface arg surfaces a hard error", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		_, err := (&suite.Generator{}).Generate(pkg, []string{"Status"},
			generator.DefaultConfig(), generator.Options{
				Output: "x.gen.go",
			})
		testkit.True(t, err != nil, "non-interface rejected")
	})
}
