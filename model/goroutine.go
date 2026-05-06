// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// goroutineIDPattern matches "goroutine N [status]:" header lines.
var goroutineIDPattern = regexp.MustCompile(`^goroutine (\d+) `)

// captureGoroutineIDs captures all goroutine IDs from the current
// process. Uses runtime.Stack with grow-to-fit (1MB start, 2x to
// 8MB cap).
func captureGoroutineIDs() map[int]struct{} {
	buf := captureAllStacks()
	return parseGoroutineIDs(buf)
}

// parseGoroutineIDs extracts goroutine IDs from runtime.Stack output.
func parseGoroutineIDs(stack []byte) map[int]struct{} {
	ids := make(map[int]struct{})
	for _, line := range strings.Split(string(stack), "\n") {
		if m := goroutineIDPattern.FindStringSubmatch(line); m != nil {
			id, err := strconv.Atoi(m[1])
			if err == nil {
				ids[id] = struct{}{}
			}
		}
	}
	return ids
}

// diffGoroutineIDs returns IDs present in end but not in start.
func diffGoroutineIDs(start, end map[int]struct{}) map[int]struct{} {
	diff := make(map[int]struct{})
	for id := range end {
		if _, ok := start[id]; !ok {
			diff[id] = struct{}{}
		}
	}
	return diff
}


// splitGoroutineStacks splits runtime.Stack output into per-goroutine
// sections keyed by goroutine ID.
func splitGoroutineStacks(stack []byte) map[int]string {
	result := make(map[int]string)
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
			id, err := strconv.Atoi(m[1])
			if err == nil {
				result[id] = full
			}
		}
	}
	return result
}

// extractStacksForIDs returns the full stack text for the given IDs.
func extractStacksForIDs(stack []byte, ids map[int]struct{}) string {
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

// CheckGoroutineLeaks runs fn and checks for goroutine leaks by
// comparing goroutine IDs before and after. This runs at the
// AssertXxxModel wrapper level (outside rapid.Check) so rapid's
// transient goroutines are quiescent at both observation points.
func CheckGoroutineLeaks(t interface {
	Helper()
	Errorf(format string, args ...any)
	Logf(format string, args ...any)
	Name() string
}, artifactDir string, fn func()) {
	t.Helper()
	startIDs := captureGoroutineIDs()
	fn()
	endIDs := captureGoroutineIDs()
	diff := diffGoroutineIDs(startIDs, endIDs)
	if len(diff) == 0 {
		return
	}
	// Re-capture full stacks and filter framework goroutines.
	fullStacks := captureAllStacks()
	leaked := filterFrameworkGoroutines(fullStacks, diff)
	if len(leaked) == 0 {
		return
	}
	// Write artifact.
	var artifactPath string
	if artifactDir != "" {
		if err := os.MkdirAll(artifactDir, 0o755); err == nil {
			name := sanitizeForFilename(t.Name()) + "-goroutines.txt"
			path := filepath.Join(artifactDir, name)
			content := extractStacksForIDs(fullStacks, leaked)
			if err := os.WriteFile(path, []byte(content), 0o644); err == nil {
				artifactPath = path
				t.Logf("goroutine stacks: %s", path)
			}
		}
	}
	if artifactPath != "" {
		t.Errorf("[liveness] AUTO-NO-GOROUTINE-LEAKS: %d goroutine(s) leaked (before: %d, after: %d)\n  goroutine stacks: %s",
			len(leaked), len(startIDs), len(endIDs), artifactPath)
	} else {
		t.Errorf("[liveness] AUTO-NO-GOROUTINE-LEAKS: %d goroutine(s) leaked (before: %d, after: %d)",
			len(leaked), len(startIDs), len(endIDs))
	}
}

// filterFrameworkGoroutines removes goroutine IDs whose stacks are
// entirely framework/runtime code. Only user-code goroutines remain.
func filterFrameworkGoroutines(stacks []byte, ids map[int]struct{}) map[int]struct{} {
	remaining := make(map[int]struct{})
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
	for _, line := range strings.Split(stack, "\n") {
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

// captureAllStacks captures all goroutine stacks with grow-to-fit
// (1MB start, 2x to 8MB cap). Returns the filled portion of the buffer.
func captureAllStacks() []byte {
	buf := make([]byte, 1<<20) // 1MB
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return buf[:n]
		}
		if len(buf) >= 8<<20 { // 8MB cap
			// Truncated — append marker and return.
			return append(buf[:n], "\n... TRUNCATED\n"...)
		}
		buf = make([]byte, len(buf)*2)
	}
}
