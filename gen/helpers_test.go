// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
)

func testdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	return filepath.Join(filepath.Dir(file), "testdata")
}

func loadTestPackage(t *testing.T, subdir string) *gen.Package {
	t.Helper()
	loader := gen.NewLoader()
	dir := filepath.Join(testdataDir(t), subdir)
	pkg, err := loader.Load(".", dir)
	testkit.NoError(t, err, "must load package")
	return pkg
}
