// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator

import (
	"fmt"
	"go/token"
	"path/filepath"
)

// SourceAttribution computes the "filename:minLine-maxLine" string for
// a set of positions. Returns the empty string when positions is empty
// or contains no position with a filename.
//
// All positions must share the same filename — the function picks the
// first non-empty filename it encounters. The line range spans the
// minimum to the maximum line number across the input set.
//
// This helper exists to eliminate the ~32-line block that was
// duplicated across stub, suite, and bench generators in gen/.
func SourceAttribution(positions []token.Position) string {
	if len(positions) == 0 {
		return ""
	}
	var filename string
	minLine, maxLine := -1, -1
	for _, p := range positions {
		if filename == "" && p.Filename != "" {
			filename = p.Filename
		}
		if p.Line == 0 {
			continue
		}
		if minLine < 0 || p.Line < minLine {
			minLine = p.Line
		}
		if p.Line > maxLine {
			maxLine = p.Line
		}
	}
	if filename == "" || minLine < 0 {
		return ""
	}
	if minLine == maxLine {
		return fmt.Sprintf("%s:%d", filepath.Base(filename), minLine)
	}
	return fmt.Sprintf("%s:%d-%d", filepath.Base(filename), minLine, maxLine)
}
