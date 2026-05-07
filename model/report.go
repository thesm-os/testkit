// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.thesmos.sh/testkit/model/trace"
)

// formatFailure produces the human-readable failure message for
// terminal and CI output. The format varies by Kind per the spec.
func formatFailure(f *Failure) string {
	var b strings.Builder

	// Header: [kind] or [REQ kind]
	writeHeader(&b, f)

	// Trace dump (Kind-gated; only present when f.Trace is non-nil).
	if len(f.Trace) > 0 {
		writeTrace(&b, f.Trace, f.StepRan.Index)
	}

	// Artifact paths.
	for _, p := range f.ArtifactPaths {
		fmt.Fprintf(&b, "  %s\n", p)
	}

	return b.String()
}

// writeHeader formats the first line: [kind] lawID at step N: message
func writeHeader(b *strings.Builder, f *Failure) {
	prefix := f.Kind.String()
	if f.REQID != "" {
		prefix = f.REQID + " " + prefix
	}

	step := f.StepRan.Index
	stepStr := fmt.Sprintf("step %d", step)
	if f.StepRan.WorkerID >= 0 {
		stepStr = fmt.Sprintf("w%d op%d", f.StepRan.WorkerID, f.StepRan.Index)
	}

	msg := "<nil>"
	if f.Err != nil {
		msg = f.Err.Error()
	}
	if f.LawID != "" {
		fmt.Fprintf(b, "[%s] %s at %s: %s\n", prefix, f.LawID, stepStr, msg)
	} else {
		fmt.Fprintf(b, "[%s] at %s: %s\n", prefix, stepStr, msg)
	}
}

// writeTrace formats the operation trace with the failing step annotated.
// For concurrent traces (ClientID >= 0), uses [wN opM] prefixes.
func writeTrace(b *strings.Builder, events []trace.Event, failingStep int) {
	// Track per-worker op index for concurrent traces.
	workerOps := make(map[int]int)
	for i, ev := range events {
		inputStr := truncateValue(ev.Inputs)
		outputStr := truncateValue(ev.Output)

		// Format step prefix: [wN opM] for concurrent, [step N] for sequential.
		var prefix string
		if ev.ClientID >= 0 {
			opIdx := workerOps[ev.ClientID]
			workerOps[ev.ClientID] = opIdx + 1
			prefix = fmt.Sprintf("w%d op%d", ev.ClientID, opIdx)
		} else {
			prefix = fmt.Sprintf("step %d", i)
		}

		var line string
		if ev.Err != nil {
			line = fmt.Sprintf("  [%s] %s(%s) -> err: %s", prefix, ev.OpName, inputStr, ev.Err.Error())
		} else {
			line = fmt.Sprintf("  [%s] %s(%s) -> %s", prefix, ev.OpName, inputStr, outputStr)
		}

		if i == failingStep {
			line += "   <-- FAILED"
		}
		fmt.Fprintln(b, line)
	}
}

// truncateValue formats a value for trace display with aggressive truncation.
func truncateValue(v any) string {
	if v == nil {
		return "<nil>"
	}

	switch val := v.(type) {
	case string:
		if len(val) > 32 {
			return fmt.Sprintf("%q...", val[:32])
		}
		return fmt.Sprintf("%q", val)
	case []byte:
		if len(val) > 64 {
			return fmt.Sprintf("[%d bytes] %x...", len(val), val[:64])
		}
		return fmt.Sprintf("%x", val)
	case error:
		s := val.Error()
		if len(s) > 32 {
			return s[:32] + "..."
		}
		return s
	case []any:
		if len(val) == 0 {
			return "[]"
		}
		if len(val) == 1 {
			return truncateValue(val[0])
		}
		var parts []string
		for _, item := range val {
			parts = append(parts, truncateValue(item))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		s := fmt.Sprintf("%+v", val)
		// Truncate structs with many fields.
		if len(s) > 120 {
			return s[:120] + "..."
		}
		return s
	}
}

// sanitizeForFilename replaces characters outside [a-zA-Z0-9._-] with _
// and truncates to 200 chars.
func sanitizeForFilename(s string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	sanitized := re.ReplaceAllString(s, "_")
	if len(sanitized) > 200 {
		sanitized = sanitized[:200]
	}
	return sanitized
}


// defaultArtifactDir is the fallback when no config or option is set.
const defaultArtifactDir = ".testkit/artifacts"

// resolveArtifactDir returns the artifact directory based on priority:
// 1. Explicit override (from WithArtifactDir option) — used as-is
// 2. Fallback: <module-root>/.testkit/artifacts/
//
// Paths are resolved relative to the module root (where go.mod lives),
// not the package dir (where go test runs). This matches .testkit.yml
// which lives at the module root.
// ResolveArtifactDir returns the artifact directory. Exported for use
// by generated code.
func ResolveArtifactDir(override string) string {
	return resolveArtifactDir(override)
}

func resolveArtifactDir(override string) string {
	if override != "" {
		return override
	}
	root := findModuleRoot()
	if root != "" {
		return filepath.Join(root, defaultArtifactDir)
	}
	return defaultArtifactDir
}

// findModuleRoot walks up from CWD looking for .testkit.yml (the
// project root config). Falls back to go.mod if .testkit.yml isn't
// found. In multi-module workspaces, .testkit.yml is the stable
// anchor — go.mod exists at every sub-module level.
func findModuleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	// First pass: look for .testkit.yml (project root).
	for d := dir; ; {
		if _, err := os.Stat(filepath.Join(d, ".testkit.yml")); err == nil {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			break
		}
		d = parent
	}
	// Fallback: look for go.mod (for projects without .testkit.yml).
	for d := dir; ; {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return ""
		}
		d = parent
	}
}
