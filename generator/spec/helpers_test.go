// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package spec_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/spec"
)

// loadFixture loads a package from generator/testdata/<name>/.
// Mirrors the helper in sibling generator subpackages.
func loadFixture(t *testing.T, name string) *generator.Package {
	t.Helper()
	pkg, err := generator.NewLoader().Load("./../testdata/"+name, "")
	testkit.NoError(t, err, "Load testdata/"+name)
	return pkg
}

// analyzeFixture runs [spec.Analyze] against testdata/<fixture>/ for
// the named interface and indexes the methods by name. The output
// path is synthesized to a sibling _test package so the tracker
// renders types through their import qualifier.
func analyzeFixture(t *testing.T, fixture, iface string) (*spec.Data, map[string]*spec.Method) {
	t.Helper()
	data, err := spec.Analyze(loadFixture(t, fixture), []string{iface},
		generator.DefaultConfig(), generator.Options{Output: fixture + "test/x.gen.go"})
	testkit.NoError(t, err, "Analyze "+iface)
	byName := make(map[string]*spec.Method, len(data.Methods))
	for i := range data.Methods {
		byName[data.Methods[i].Name] = &data.Methods[i]
	}
	return data, byName
}
