// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sentinel_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
)

// loadBasic loads the testdata/basic fixture used across analyze,
// generator, and template tests.
func loadBasic(t *testing.T) *generator.Package {
	t.Helper()
	loader := generator.NewLoader()
	pkg, err := loader.Load("./../testdata/basic", "")
	testkit.NoError(t, err, "Load testdata/basic")
	return pkg
}
