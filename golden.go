// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var updateGolden = flag.Bool("update", false, "update golden files")

// ShouldUpdate reports whether the -update flag was passed to go test.
//
//	go test -update ./...
func ShouldUpdate() bool {
	return *updateGolden
}

// Scrubber transforms golden file content before comparison. Use scrubbers
// to remove non-deterministic values (timestamps, hashes, run IDs) so that
// golden comparisons are stable across runs.
type Scrubber func([]byte) []byte

// AssertGolden compares got against the golden file at
// testdata/golden/<file> relative to the test's package directory.
//
// Behavior:
//   - File missing + no -update: fail (cite -update flag)
//   - File missing + -update: write file, pass
//   - Content matches: pass
//   - Content differs + no -update: fail with [cmp.Diff]
//   - Content differs + -update: overwrite file, pass
//
// Scrubbers are applied left-to-right to got before comparison.
//
//	testkit.AssertGolden(t, "response.json", body, testkit.ScrubTimestamps())
func AssertGolden(tb testing.TB, file string, got []byte, scrubbers ...Scrubber) {
	tb.Helper()
	assertGolden(tb, file, got, *updateGolden, scrubbers...)
}

// assertGolden is the testable core of [AssertGolden]. The update parameter
// is injected so tests can exercise both the update and compare paths without
// needing the -update flag.
func assertGolden(tb testing.TB, file string, got []byte, update bool, scrubbers ...Scrubber) {
	tb.Helper()
	for _, s := range scrubbers {
		got = s(got)
	}

	path := filepath.Join("testdata", "golden", file)

	if update {
		dir := filepath.Dir(path)
		err := os.MkdirAll(dir, 0o750)
		if err != nil {
			tb.Fatalf("AssertGolden: mkdir %s: %v", dir, err)
			return
		}
		err = os.WriteFile(path, got, 0o644) //nolint:gosec // golden file permissions are fine
		if err != nil {
			tb.Fatalf("AssertGolden: write %s: %v", path, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			tb.Fatalf("AssertGolden: %s does not exist — run with -update to create it", path)
			return
		}
		tb.Fatalf("AssertGolden: read %s: %v", path, err)
		return
	}

	if diff := cmp.Diff(string(want), string(got)); diff != "" {
		tb.Fatalf("AssertGolden: %s (-want +got)\n%s", file, diff)
	}
}

// --- Pre-built scrubbers ---

var (
	timestampRe = regexp.MustCompile(
		`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:?\d{2})?`,
	)
	hashRe  = regexp.MustCompile(`\b[0-9a-f]{32,128}\b`)
	runIDRe = regexp.MustCompile(`run_[a-z0-9]{16}`)
)

// ScrubTimestamps returns a [Scrubber] that replaces ISO-8601 and RFC-3339
// timestamps with "SCRUBBED_TS".
func ScrubTimestamps() Scrubber {
	return func(b []byte) []byte {
		return timestampRe.ReplaceAll(b, []byte("SCRUBBED_TS"))
	}
}

// ScrubHashes returns a [Scrubber] that replaces hex digests (32–128 chars)
// with "SCRUBBED_HASH".
func ScrubHashes() Scrubber {
	return func(b []byte) []byte {
		return hashRe.ReplaceAll(b, []byte("SCRUBBED_HASH"))
	}
}

// ScrubRunIDs returns a [Scrubber] that replaces run_[a-z0-9]{16} patterns
// with "SCRUBBED_RUN".
func ScrubRunIDs() Scrubber {
	return func(b []byte) []byte {
		return runIDRe.ReplaceAll(b, []byte("SCRUBBED_RUN"))
	}
}

// ScrubJSONFields returns a [Scrubber] that replaces the values of named
// JSON fields with "SCRUBBED". It uses a simple regex approach — not a full
// JSON parser — so it works on both valid and nearly-valid JSON.
//
//	testkit.ScrubJSONFields("created_at", "token")
func ScrubJSONFields(fields ...string) Scrubber {
	patterns := make([]*regexp.Regexp, len(fields))
	for i, f := range fields {
		patterns[i] = regexp.MustCompile(
			`("` + regexp.QuoteMeta(f) + `"\s*:\s*)("(?:[^"\\]|\\.)*"|[\w.+-]+)`,
		)
	}
	return func(b []byte) []byte {
		for _, p := range patterns {
			b = p.ReplaceAll(b, []byte(`${1}"SCRUBBED"`))
		}
		return b
	}
}
