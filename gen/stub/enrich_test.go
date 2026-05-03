// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
	"go.thesmos.sh/testkit/gen/stub"
)

func analyzeDirectives(t *testing.T) (*stub.Data, *gen.Package) {
	t.Helper()
	pkg := loadTestPackage(t, "directives")
	data, err := stub.Analyze(pkg, []string{"Store"}, gen.DefaultConfig(), gen.Options{
		Output: "storetest/store_stub.gen.go",
	})
	testkit.NoError(t, err, "must analyze")
	return data, pkg
}

func TestEnrich(t *testing.T) {
	t.Parallel()

	t.Run("errors populates sentinels from args", func(t *testing.T) {
		t.Parallel()
		data, pkg := analyzeDirectives(t)
		err := stub.Enrich(data, pkg)
		testkit.NoError(t, err, "must enrich")

		var get *stub.MethodData
		for _, m := range data.Interfaces[0].Methods {
			if m.Name == "Get" {
				get = m
			}
		}
		testkit.Len(t, get.Sentinels, 2, "Get must have 2 sentinels")
		testkit.Equal(t, get.Sentinels[0].VarName, "ErrNotFound", "first sentinel var")
		testkit.Equal(t, get.Sentinels[0].ShortName, "NotFound", "first sentinel short name")
		testkit.Assert(t, get.Sentinels[0].Qualified).Contains("ErrNotFound", "must be qualified")
		testkit.Equal(t, get.Sentinels[1].VarName, "ErrConflict", "second sentinel var")
	})

	t.Run("errors Put has one sentinel", func(t *testing.T) {
		t.Parallel()
		data, pkg := analyzeDirectives(t)
		err := stub.Enrich(data, pkg)
		testkit.NoError(t, err, "must enrich")

		var put *stub.MethodData
		for _, m := range data.Interfaces[0].Methods {
			if m.Name == "Put" {
				put = m
			}
		}
		testkit.Len(t, put.Sentinels, 1, "Put must have 1 sentinel")
		testkit.Equal(t, put.Sentinels[0].VarName, "ErrConflict", "sentinel")
	})

	t.Run("errors Delete has no sentinels", func(t *testing.T) {
		t.Parallel()
		data, pkg := analyzeDirectives(t)
		err := stub.Enrich(data, pkg)
		testkit.NoError(t, err, "must enrich")

		var del *stub.MethodData
		for _, m := range data.Interfaces[0].Methods {
			if m.Name == "Delete" {
				del = m
			}
		}
		testkit.Len(t, del.Sentinels, 0, "Delete must have no sentinels")
	})

	t.Run("integration-only sets Skip", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Get")
		d := gen.Directive{Name: "integration-only"}
		err := stub.EnrichIntegrationOnlyForTest(m, d, nil)
		testkit.NoError(t, err, "must succeed")
		testkit.True(t, m.Skip, "must set Skip")
	})

	t.Run("deprecated sets replacement method", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Get")
		d := gen.Directive{Name: "deprecated", Args: []string{"PutBatch"}}
		err := stub.EnrichDeprecatedForTest(m, d, nil)
		testkit.NoError(t, err, "must succeed")
		testkit.Equal(t, m.Deprecated, "PutBatch", "must set replacement")
	})

	t.Run("deprecated no args returns error", func(t *testing.T) {
		t.Parallel()
		m := buildMethodData(t, "Get")
		d := gen.Directive{Name: "deprecated"}
		err := stub.EnrichDeprecatedForTest(m, d, nil)
		testkit.Error(t, err, "must fail with no args")
	})

	t.Run("errors with invalid sentinel returns error", func(t *testing.T) {
		t.Parallel()
		data, pkg := analyzeDirectives(t)
		// Inject a directive with a nonexistent sentinel.
		for _, m := range data.Interfaces[0].Methods {
			if m.Name == "Delete" {
				m.Directives = append(m.Directives, gen.Directive{
					Name: "errors",
					Args: []string{"ErrNonexistent"},
				})
			}
		}
		err := stub.Enrich(data, pkg)
		testkit.Error(t, err, "must fail for invalid sentinel")
	})

	t.Run("errors with no args returns error", func(t *testing.T) {
		t.Parallel()
		data, pkg := analyzeDirectives(t)
		for _, m := range data.Interfaces[0].Methods {
			if m.Name == "Delete" {
				m.Directives = append(m.Directives, gen.Directive{
					Name: "errors",
				})
			}
		}
		err := stub.Enrich(data, pkg)
		testkit.Error(t, err, "must fail with no args")
	})

	t.Run("errors with same-package qualifier is unqualified", func(t *testing.T) {
		t.Parallel()
		// When output is in the same package, the qualifier is empty
		// and sentinel names are unqualified.
		pkg := loadTestPackage(t, "directives")
		data, err := stub.Analyze(pkg, []string{"Store"}, gen.DefaultConfig(), gen.Options{
			Output: "store_stub.gen.go", // same directory = same package
		})
		testkit.NoError(t, err, "must analyze")
		enrichErr := stub.Enrich(data, pkg)
		testkit.NoError(t, enrichErr, "must enrich")

		var get *stub.MethodData
		for _, m := range data.Interfaces[0].Methods {
			if m.Name == "Get" {
				get = m
			}
		}
		// When output is same package, sentinel should not be qualified.
		testkit.Equal(t, get.Sentinels[0].Qualified, "ErrNotFound", "same-package must be unqualified")
	})

	t.Run("skips directives not relevant to stub", func(t *testing.T) {
		t.Parallel()
		data, pkg := analyzeDirectives(t)
		// idempotent is a suite directive, not a stub directive.
		// The enricher skips it — unknown-directive validation happens
		// in the generator pipeline via directive.Registry.Validate().
		data.Interfaces[0].Methods[0].Directives = append(
			data.Interfaces[0].Methods[0].Directives,
			gen.Directive{Name: "idempotent"},
		)
		err := stub.Enrich(data, pkg)
		testkit.NoError(t, err, "directives not relevant to stub must be skipped")
	})
}
