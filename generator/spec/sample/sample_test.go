// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sample_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/sample"
)

// loadSamplerData runs spec.Analyze against testdata/basic's
// Sampler interface, returning the package and the resulting Data.
// Used as the common setup for every sample test.
func loadSamplerData(t *testing.T) (*generator.Package, *spec.Data) {
	t.Helper()
	pkg, err := generator.NewLoader().Load("./../../testdata/basic", "")
	testkit.NoError(t, err, "Load testdata/basic")
	data, err := spec.Analyze(pkg, []string{"Sampler"},
		generator.DefaultConfig(), generator.Options{Output: "samplertest/sampler.gen.go"})
	testkit.NoError(t, err, "Analyze")
	return pkg, data
}

// indexMethods returns methods keyed by name for table-driven asserts.
func indexMethods(ms []spec.Method) map[string]spec.Method {
	out := make(map[string]spec.Method, len(ms))
	for _, m := range ms {
		out[m.Name] = m
	}
	return out
}

// setDirectives replaces the directive list for one method by name.
// Used by negative tests to inject malformed directives without
// editing the fixture.
func setDirectives(data *spec.Data, methodName string, dirs ...directive.Directive) {
	for i := range data.Methods {
		if data.Methods[i].Name == methodName {
			data.Methods[i].Directives = dirs
			return
		}
	}
}

// mustPayload retrieves the sample payload or fails the test.
func mustPayload(t *testing.T, m spec.Method) sample.Payload {
	t.Helper()
	got, ok := spec.Get[sample.Payload](m.Attachments, directive.Sample)
	testkit.True(t, ok, "sample payload attached for "+m.Name)
	return got
}

func TestConsume(t *testing.T) {
	t.Parallel()

	t.Run("happy path: per-param sample funcs attach to method", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadSamplerData(t)
		testkit.NoError(t, spec.Enrich(data, pkg), "Enrich")
		byName := indexMethods(data.Methods)
		// Lookup has one non-ctx param (key string) → one sample.
		testkit.Equal(t, mustPayload(t, byName["Lookup"]).Calls,
			[]string{"SampleKey"}, "Lookup sample call")
		// Apply has two non-ctx params → two samples.
		testkit.Equal(t, mustPayload(t, byName["Apply"]).Calls,
			[]string{"SampleKey", "SampleItem"}, "Apply sample calls")
	})

	t.Run("rejects when arg count != non-ctx param count", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadSamplerData(t)
		setDirectives(data, "Apply",
			directive.Directive{Name: directive.Sample, Args: []string{"SampleKey"}})
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "count mismatch must error")
		testkit.Assert(t, err.Error()).Contains("expects 2 arg(s)", "diagnostic")
	})

	t.Run("rejects when named sample func is missing", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadSamplerData(t)
		setDirectives(data, "Lookup",
			directive.Directive{Name: directive.Sample, Args: []string{"DoesNotExist"}})
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "missing func must error")
		testkit.Assert(t, err.Error()).Contains("not found", "diagnostic")
	})

	t.Run("rejects when sample func has wrong return type", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadSamplerData(t)
		// Lookup's key is a string; SampleItem returns Item — type mismatch.
		setDirectives(data, "Lookup",
			directive.Directive{Name: directive.Sample, Args: []string{"SampleItem"}})
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "type mismatch must error")
		testkit.Assert(t, err.Error()).Contains("result 0", "diagnostic names mismatched result")
	})

	t.Run("cross-package: qualified arg loads remote pkg + tracker registers import", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadSamplerData(t)
		const remote = "go.thesmos.sh/testkit/generator/testdata/storage"
		setDirectives(data, "Lookup",
			directive.Directive{Name: directive.Sample, Args: []string{remote + ".SampleKey"}})
		testkit.NoError(t, spec.Enrich(data, pkg), "Enrich")
		got := mustPayload(t, indexMethods(data.Methods)["Lookup"])
		testkit.Equal(t, got.Calls, []string{"storage.SampleKey"},
			"alias.FuncName uses tracker base name")
	})

	t.Run("cross-package: nonexistent remote pkg surfaces a load error", func(t *testing.T) {
		t.Parallel()
		pkg, data := loadSamplerData(t)
		setDirectives(data, "Lookup",
			directive.Directive{Name: directive.Sample, Args: []string{"go.thesmos.sh/does/not/exist.X"}})
		err := spec.Enrich(data, pkg)
		testkit.True(t, err != nil, "missing remote pkg must error")
	})
}
