// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"go.thesmos.sh/testkit/concurrency"
)

// goroutineIDPattern matches "goroutine N [status]:" header lines.
var goroutineIDPattern = regexp.MustCompile(`^goroutine (\d+) `)

// splitGoroutineStacks splits runtime.Stack output into per-goroutine
// sections keyed by goroutine ID. Model needs this beyond what
// [concurrency.CaptureGoroutineIDs] returns because the leak filter
// inspects per-goroutine call frames to drop framework-only stacks.
func splitGoroutineStacks(stack []byte) map[uint64]string {
	result := make(map[uint64]string)
	sections := strings.Split(string(stack), "\ngoroutine ")
	for i, section := range sections {
		var full string
		if i == 0 {
			// First section starts with "goroutine " (no leading \n).
			if !strings.HasPrefix(section, "goroutine ") {
				continue
			}
			full = section
		} else {
			full = "goroutine " + section
		}
		if m := goroutineIDPattern.FindStringSubmatch(full); m != nil {
			id, err := strconv.ParseUint(m[1], 10, 64)
			if err == nil {
				result[id] = full
			}
		}
	}
	return result
}

// extractStacksForIDs returns the full stack text for the given IDs.
func extractStacksForIDs(stack []byte, ids map[uint64]struct{}) string {
	goroutines := splitGoroutineStacks(stack)
	var b strings.Builder
	for id := range ids {
		if g, ok := goroutines[id]; ok {
			b.WriteString(g)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// CheckGoroutineLeaks runs fn and reports goroutine leaks via t.Errorf.
// Capture and diff use [concurrency.CaptureGoroutineIDs] /
// [concurrency.DiffGoroutineIDs]; model adds two layers on top:
//
//   - Framework-frame filtering — goroutines whose stacks consist
//     entirely of runtime/testing/rapid frames are dropped (rapid's
//     transient workers, runtime poll, etc.).
//   - Artifact emission — leaked goroutines' full stacks are written
//     to <artifactDir>/<test>-goroutines.txt for later inspection.
//
// This runs at the AssertXxxModel wrapper level (outside rapid.Check)
// so rapid's transient goroutines are quiescent at both observation
// points.
func CheckGoroutineLeaks(t interface {
	Helper()
	Errorf(format string, args ...any)
	Logf(format string, args ...any)
	Name() string
}, artifactDir string, fn func(),
) {
	t.Helper()
	startIDs := concurrency.CaptureGoroutineIDs()
	fn()
	endIDs := concurrency.CaptureGoroutineIDs()
	leaked := concurrency.DiffGoroutineIDs(startIDs, endIDs)
	if len(leaked) == 0 {
		return
	}
	leakedSet := make(map[uint64]struct{}, len(leaked))
	for _, id := range leaked {
		leakedSet[id] = struct{}{}
	}
	// Re-capture full stacks and filter framework goroutines.
	fullStacks := concurrency.CaptureGoroutineStacks()
	userLeaked := filterFrameworkGoroutines(fullStacks, leakedSet)
	if len(userLeaked) == 0 {
		return
	}
	// Write artifact.
	var artifactPath string
	if artifactDir != "" {
		err := os.MkdirAll(artifactDir, 0o750) //nolint:gosec // test artifacts
		if err == nil {
			name := sanitizeForFilename(t.Name()) + "-goroutines.txt"
			path := filepath.Join(artifactDir, name)
			content := extractStacksForIDs(fullStacks, userLeaked)
			err := os.WriteFile(path, []byte(content), 0o600) //nolint:gosec // test artifacts
			if err == nil {
				artifactPath = path
				t.Logf("goroutine stacks: %s", path)
			}
		}
	}
	if artifactPath != "" {
		t.Errorf(
			"[liveness] AUTO-NO-GOROUTINE-LEAKS: %d goroutine(s) leaked (before: %d, after: %d)\n  goroutine stacks: %s",
			len(userLeaked),
			len(startIDs),
			len(endIDs),
			artifactPath,
		)
	} else {
		t.Errorf("[liveness] AUTO-NO-GOROUTINE-LEAKS: %d goroutine(s) leaked (before: %d, after: %d)",
			len(userLeaked), len(startIDs), len(endIDs))
	}
}

// filterFrameworkGoroutines removes goroutine IDs whose stacks are
// entirely framework/runtime code. Only user-code goroutines remain.
func filterFrameworkGoroutines(stacks []byte, ids map[uint64]struct{}) map[uint64]struct{} {
	remaining := make(map[uint64]struct{})
	sections := splitGoroutineStacks(stacks)
	for id := range ids {
		if section, ok := sections[id]; ok {
			if !isFrameworkOnlyStack(section) {
				remaining[id] = struct{}{}
			}
		}
	}
	return remaining
}

// isFrameworkOnlyStack returns true if all function frames in the
// stack belong to framework/runtime packages.
func isFrameworkOnlyStack(stack string) bool {
	for line := range strings.SplitSeq(stack, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "goroutine ") {
			continue
		}
		if strings.Contains(line, ".go:") {
			continue // file path line
		}
		// Function call line — check if it's framework.
		if !isFrameworkFrame(line) {
			return false
		}
	}
	return true
}

var frameworkFramePrefixes = []string{
	"runtime.",
	"runtime/",
	"pgregory.net/rapid",
	"testing.",
	"go.thesmos.sh/testkit/model.",
	"go.thesmos.sh/testkit/concurrency.",
	"internal/poll.",
}

func isFrameworkFrame(line string) bool {
	for _, prefix := range frameworkFramePrefixes {
		if strings.Contains(line, prefix) {
			return true
		}
	}
	return false
}
