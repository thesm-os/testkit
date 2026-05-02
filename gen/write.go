// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	newFileDiffFmt = "--- %s (does not exist)\n+++ %s (generated)\n(new file)"
	diffHeaderOld  = "--- %s\n"
	diffHeaderNew  = "+++ %s (generated)\n"
	diffRemoved    = "-%s\n"
	diffAdded      = "+%s\n"
)

// WriteResult writes all files in a [Result] to disk. Paths are
// resolved relative to workDir. Directories are created as needed.
//
// When check is true, it compares each file against existing content
// and returns an error with unified diffs for any files that differ
// (dry-run mode for CI).
func WriteResult(result *Result, workDir string, check bool) error {
	var diffs []string
	for _, f := range result.Files {
		absPath := filepath.Join(workDir, f.Path)
		if check {
			existing, err := os.ReadFile(absPath)
			if err != nil {
				if os.IsNotExist(err) {
					diffs = append(diffs,
						fmt.Sprintf(newFileDiffFmt, f.Path, f.Path))
					continue
				}
				return fmt.Errorf("read %s: %w", f.Path, err)
			}
			if !bytes.Equal(existing, f.Content) {
				diffs = append(diffs, simpleDiff(f.Path, string(existing), string(f.Content)))
			}
			continue
		}

		dir := filepath.Dir(absPath)
		err := os.MkdirAll(dir, 0o750)
		if err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
		err = os.WriteFile(absPath, f.Content, 0o644) //nolint:gosec // generated file permissions
		if err != nil {
			return fmt.Errorf("write %s: %w", f.Path, err)
		}
	}
	if len(diffs) > 0 {
		return fmt.Errorf("%d file(s) would change:\n\n%s", len(diffs), strings.Join(diffs, "\n\n"))
	}
	return nil
}

// simpleDiff produces a basic unified-style diff between old and
// updated content.
func simpleDiff(path, old, updated string) string {
	oldLines := strings.Split(old, "\n")
	newLines := strings.Split(updated, "\n")

	var b strings.Builder
	fmt.Fprintf(&b, diffHeaderOld, path)
	fmt.Fprintf(&b, diffHeaderNew, path)

	maxLen := max(len(oldLines), len(newLines))
	for i := range maxLen {
		var o, n string
		if i < len(oldLines) {
			o = oldLines[i]
		}
		if i < len(newLines) {
			n = newLines[i]
		}
		if o != n {
			if i < len(oldLines) {
				fmt.Fprintf(&b, diffRemoved, o)
			}
			if i < len(newLines) {
				fmt.Fprintf(&b, diffAdded, n)
			}
		}
	}
	return b.String()
}
