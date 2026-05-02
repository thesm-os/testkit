// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
)

func TestWriteResult(t *testing.T) {
	t.Parallel()

	t.Run("creates directories and writes files", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		result := &gen.Result{
			Files: []gen.OutputFile{
				{Path: "sub/file.go", Content: []byte("package sub\n")},
			},
		}
		err := gen.WriteResult(result, dir, false)
		testkit.NoError(t, err, "write must succeed")

		data, readErr := os.ReadFile(filepath.Join(dir, "sub", "file.go"))
		testkit.NoError(t, readErr, "must read written file")
		testkit.Equal(t, string(data), "package sub\n", "content must match")
	})

	t.Run("overwrites existing files", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		// Write initial content.
		err := os.MkdirAll(filepath.Join(dir, "sub"), 0o750)
		testkit.NoError(t, err, "mkdir")
		err = os.WriteFile(filepath.Join(dir, "sub", "file.go"), []byte("old"), 0o644) //nolint:gosec
		testkit.NoError(t, err, "write old")

		result := &gen.Result{
			Files: []gen.OutputFile{
				{Path: "sub/file.go", Content: []byte("new")},
			},
		}
		err = gen.WriteResult(result, dir, false)
		testkit.NoError(t, err, "overwrite must succeed")

		data, readErr := os.ReadFile(filepath.Join(dir, "sub", "file.go"))
		testkit.NoError(t, readErr, "must read")
		testkit.Equal(t, string(data), "new", "must contain new content")
	})

	t.Run("check mode passes when identical", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		content := []byte("package foo\n")

		err := os.WriteFile(filepath.Join(dir, "file.go"), content, 0o644) //nolint:gosec
		testkit.NoError(t, err, "setup")

		result := &gen.Result{
			Files: []gen.OutputFile{
				{Path: "file.go", Content: content},
			},
		}
		err = gen.WriteResult(result, dir, true)
		testkit.NoError(t, err, "check must pass when identical")
	})

	t.Run("check mode fails with diff when different", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		err := os.WriteFile(filepath.Join(dir, "file.go"), []byte("old\n"), 0o644) //nolint:gosec
		testkit.NoError(t, err, "setup")

		result := &gen.Result{
			Files: []gen.OutputFile{
				{Path: "file.go", Content: []byte("new\n")},
			},
		}
		err = gen.WriteResult(result, dir, true)
		testkit.Error(t, err, "check must fail when different")
		testkit.Assert(t, err.Error()).
			Contains("would change", "must mention changes").
			Contains("file.go", "must mention file")
	})

	t.Run("check mode reports new files", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		result := &gen.Result{
			Files: []gen.OutputFile{
				{Path: "new_file.go", Content: []byte("package foo\n")},
			},
		}
		err := gen.WriteResult(result, dir, true)
		testkit.Error(t, err, "check must fail for new file")
		testkit.Assert(t, err.Error()).Contains("does not exist", "must report missing file")
	})

	t.Run("writes multiple files", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		result := &gen.Result{
			Files: []gen.OutputFile{
				{Path: "a.go", Content: []byte("package a\n")},
				{Path: "b.go", Content: []byte("package b\n")},
			},
		}
		err := gen.WriteResult(result, dir, false)
		testkit.NoError(t, err, "must write both files")

		_, err1 := os.Stat(filepath.Join(dir, "a.go"))
		_, err2 := os.Stat(filepath.Join(dir, "b.go"))
		testkit.NoError(t, err1, "a.go must exist")
		testkit.NoError(t, err2, "b.go must exist")
	})
}
