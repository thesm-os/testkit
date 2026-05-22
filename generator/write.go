// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteResult writes every file in result to disk under workDir.
// Directories are created as needed.
//
// When check is true (the --check / dry-run flag), WriteResult does
// not modify disk. It compares each file against the existing content
// and returns an error with unified-style diffs for any mismatch.
// New files (no existing version) count as a diff.
//
// Returns nil when every file matches in check mode, or after every
// file is successfully written in normal mode.
func WriteResult(result *Result, workDir string, check bool) error {
	var diffs []string
	for _, f := range result.Files {
		absPath := filepath.Join(workDir, f.Path)
		if check {
			existing, err := os.ReadFile(absPath)
			if err != nil {
				if os.IsNotExist(err) {
					diffs = append(diffs, fmt.Sprintf(
						"--- %s (does not exist)\n+++ %s (generated)\n(new file)",
						f.Path, f.Path,
					))
					continue
				}
				return fmt.Errorf("read %s: %w", f.Path, err)
			}
			if !bytes.Equal(existing, f.Content) {
				diffs = append(diffs, simpleDiff(f.Path, string(existing), string(f.Content)))
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(absPath), 0o750); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(absPath), err)
		}
		if err := os.WriteFile(absPath, f.Content, 0o644); err != nil { //nolint:gosec // generated file permissions
			return fmt.Errorf("write %s: %w", f.Path, err)
		}
	}
	if len(diffs) > 0 {
		return fmt.Errorf("%d file(s) would change:\n\n%s",
			len(diffs), strings.Join(diffs, "\n\n"))
	}
	return nil
}

// simpleDiff produces a unified-style diff between old and updated
// content. Used by [WriteResult] in check mode.
func simpleDiff(path, oldContent, newContent string) string {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	var b strings.Builder
	fmt.Fprintf(&b, "--- %s\n", path)
	fmt.Fprintf(&b, "+++ %s (generated)\n", path)

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
				fmt.Fprintf(&b, "-%s\n", o)
			}
			if i < len(newLines) {
				fmt.Fprintf(&b, "+%s\n", n)
			}
		}
	}
	return b.String()
}
