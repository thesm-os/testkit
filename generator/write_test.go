// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
)

func TestWriteResult(t *testing.T) {
	t.Parallel()

	t.Run("write mode creates files and intermediate directories", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		nestedPath := filepath.Join("sub", "dir", "x.gen.go")
		result := &generator.Result{
			Files: []generator.OutputFile{
				{Path: nestedPath, Content: []byte("package x\n")},
				{Path: "y.gen.go", Content: []byte("package y\n")},
			},
		}
		testkit.NoError(t, generator.WriteResult(result, dir, false), "write")

		got, err := os.ReadFile(filepath.Join(dir, nestedPath))
		testkit.NoError(t, err, "read nested file")
		testkit.Equal(t, string(got), "package x\n", "nested content matches")

		got, err = os.ReadFile(filepath.Join(dir, "y.gen.go"))
		testkit.NoError(t, err, "read top-level file")
		testkit.Equal(t, string(got), "package y\n", "top-level content matches")
	})

	t.Run("check mode passes when on-disk content matches", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "x.gen.go")
		testkit.NoError(t, os.WriteFile(path, []byte("package x\n"), 0o600), "seed")
		result := &generator.Result{
			Files: []generator.OutputFile{
				{Path: "x.gen.go", Content: []byte("package x\n")},
			},
		}
		testkit.NoError(t, generator.WriteResult(result, dir, true),
			"check passes when content matches")
	})

	t.Run("check mode reports diff when content differs", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "x.gen.go")
		testkit.NoError(t, os.WriteFile(path, []byte("package x\nold\n"), 0o600), "seed")
		result := &generator.Result{
			Files: []generator.OutputFile{
				{Path: "x.gen.go", Content: []byte("package x\nnew\n")},
			},
		}
		err := generator.WriteResult(result, dir, true)
		testkit.True(t, err != nil, "diff is reported as error")
		testkit.Assert(t, err.Error()).
			Contains("1 file(s) would change", "summary line").
			Contains("-old", "deleted line").
			Contains("+new", "added line")
	})

	t.Run("check mode reports new file when target does not exist", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		result := &generator.Result{
			Files: []generator.OutputFile{
				{Path: "missing.gen.go", Content: []byte("package x\n")},
			},
		}
		err := generator.WriteResult(result, dir, true)
		testkit.True(t, err != nil, "new file flagged")
		testkit.Assert(t, err.Error()).Contains("(new file)", "diff carries new-file marker")
	})
}
