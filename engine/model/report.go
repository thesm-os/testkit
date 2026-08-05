// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.thesmos.sh/testkit/core/trace"
)

// formatFailure produces the human-readable failure message for
// terminal and CI output. The format varies by Kind per the spec:
// after the [kind]-prefixed header, a per-kind reporter adds
// kind-specific context (structural callout, semantic divergence
// framing, invariant citation, or goroutine stacks for liveness).
// Trace dump and artifact paths render last when present.
func formatFailure(f *Failure) string {
	var b strings.Builder

	// Header: [kind] or [REQ kind]
	writeHeader(&b, f)

	// Per-kind context (Structural / Semantic / Invariant /
	// Liveness). Unclassified kinds skip the dispatch.
	switch f.Kind {
	case FailureStructural:
		writeStructural(&b, f)
	case FailureSemantic:
		writeSemantic(&b, f)
	case FailureInvariant:
		writeInvariant(&b, f)
	case FailureLiveness:
		writeLiveness(&b, f)
	case FailureUnclassified:
		// no per-kind framing; header + trace + artifacts only.
	}

	// Trace dump (Kind-gated; only present when f.Trace is non-nil).
	if len(f.Trace) > 0 {
		writeTrace(&b, f.Trace, f.StepRan.Index)
	}

	// Artifact paths.
	for _, p := range f.ArtifactPaths {
		fmt.Fprintf(&b, "  %s\n", p) //nolint:forbidigo // strings.Builder
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

	msg := nilStr
	if f.Err != nil {
		msg = f.Err.Error()
	}
	if f.LawID != "" {
		fmt.Fprintf(b, "[%s] %s at %s: %s\n", prefix, f.LawID, stepStr, msg) //nolint:forbidigo // strings.Builder
	} else {
		fmt.Fprintf(b, "[%s] at %s: %s\n", prefix, stepStr, msg) //nolint:forbidigo // strings.Builder
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
			line = fmt.Sprintf("  [%s] %s(%s) -> err: %s", prefix, ev.Method, inputStr, ev.Err.Error())
		} else {
			line = fmt.Sprintf("  [%s] %s(%s) -> %s", prefix, ev.Method, inputStr, outputStr)
		}

		if i == failingStep {
			line += "   <-- FAILED"
		}
		fmt.Fprintln(b, line) //nolint:forbidigo // strings.Builder
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
		return hex.EncodeToString(val)
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

func walkUpFor(dir, filename string) string {
	for d := dir; ; {
		_, statErr := os.Stat(filepath.Join(d, filename))
		if statErr == nil {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return ""
		}
		d = parent
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

// ResolveArtifactDir returns the artifact directory based on priority:
//
//  1. Explicit override (from [WithArtifactDir] option) — used as-is.
//  2. Fallback: `<module-root>/.testkit/artifacts/`.
//
// Paths resolve relative to the module root (where .testkit.yaml or
// go.mod lives), not the package dir (where `go test` runs). This
// matches .testkit.yaml which lives at the module root.
//
// Exported so generator-emitted code can resolve consistently. The result is
// never empty: with no override and no discoverable root it falls back to the
// relative default, so callers never have to handle "nowhere to write".
func ResolveArtifactDir(override string) string {
	if override != "" {
		return override
	}
	root := findModuleRoot()
	if root != "" {
		return filepath.Join(root, defaultArtifactDir)
	}
	return defaultArtifactDir
}

// findModuleRoot walks up from CWD looking for .testkit.yaml (the
// project root config). Falls back to go.mod if it isn't found.
//
// In a multi-module workspace the config file is the stable anchor and go.mod
// is not: every sub-module has its own go.mod, so the fallback resolves to the
// sub-module root rather than the project root, and artifacts scatter one
// directory per module instead of collecting at the top.
func findModuleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	// First pass: look for the project config.
	if found := walkUpFor(dir, ".testkit.yaml"); found != "" {
		return found
	}
	// Fallback: look for go.mod (for projects without a config file).
	return walkUpFor(dir, "go.mod")
}
