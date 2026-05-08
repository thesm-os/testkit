// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enum_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
)

// loadFixture loads a package from generator/testdata/<name>/.
// Tests use this against shared fixtures in the rebuild's testdata
// tree — parameterized so adding a new fixture (e.g. "wirebreak")
// doesn't require copying the boilerplate.
func loadFixture(t *testing.T, name string) *generator.Package {
	t.Helper()
	pkg, err := generator.NewLoader().Load("./../testdata/"+name, "")
	testkit.NoError(t, err, "Load testdata/"+name)
	return pkg
}
