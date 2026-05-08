// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package golden

import (
	"encoding/json"
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
// Use [AssertGoldenAt] when the golden lives anywhere else.
//
// Pass [ShouldUpdate]() for the update parameter to enable flag-driven
// golden file creation/update via `go test -update`.
//
// Behavior:
//   - File missing + update=false: fail (cite -update flag)
//   - File missing + update=true: write file, pass
//   - Content matches: pass
//   - Content differs + update=false: fail with [cmp.Diff]
//   - Content differs + update=true: overwrite file, pass
//
// Scrubbers are applied left-to-right to got before comparison.
//
//	golden.AssertGolden(t, "response.json", body, golden.ShouldUpdate(), golden.ScrubTimestamps())
func AssertGolden(tb testing.TB, file string, got []byte, update bool, scrubbers ...Scrubber) {
	tb.Helper()
	AssertGoldenAt(tb, filepath.Join("testdata", "golden", file), got, update, scrubbers...)
}

// AssertGoldenAt is the path-explicit form of [AssertGolden]. The path
// argument is taken verbatim relative to the test's working directory —
// no testdata/golden/ prefix is prepended. Use this when the golden
// file lives outside the canonical convention, e.g. fixture-style
// goldens that need to sit next to the source they describe.
//
// Behavior, scrubber semantics, and the -update flag handling match
// [AssertGolden].
func AssertGoldenAt(tb testing.TB, path string, got []byte, update bool, scrubbers ...Scrubber) {
	tb.Helper()
	for _, s := range scrubbers {
		got = s(got)
	}

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

	if diff := Compare(want, got); diff != "" {
		tb.Fatalf("AssertGolden: %s (-want +got)\n%s", path, diff)
	}
}

// Compare returns the [cmp.Diff] between want and got after running
// got through scrubbers (left-to-right). Returns the empty string when
// content matches.
//
// Useful in non-test contexts (e.g. a generator's --check mode, a
// post-build verifier) that want the comparison logic without
// [testing.TB] integration.
func Compare(want, got []byte, scrubbers ...Scrubber) string {
	for _, s := range scrubbers {
		got = s(got)
	}
	return cmp.Diff(string(want), string(got))
}

// AssertGoldenJSONField asserts that the JSON file at path, when
// parsed as a top-level object and keyed by field, matches got.
// Use this when one golden file holds multiple independent slices
// (e.g. one entry per type) so that per-slice tests can compare
// only their own data without the others' contents in the diff.
//
// The path argument is taken verbatim relative to the test's
// working directory — same convention as [AssertGoldenAt].
//
// Behavior:
//   - File missing + update=false: fail (cite -update flag)
//   - File missing + update=true: write `{"<field>": <got>}`
//   - Field missing + update=false: fail with diagnostic naming the field
//   - Field missing + update=true: add the field, preserving siblings
//   - Field present + matches got: pass
//   - Field present + differs + update=false: fail with [cmp.Diff] over the field's slice
//   - Field present + differs + update=true: replace just the field; siblings preserved
//
// Comparison is structural (both sides re-marshaled with the same
// indent) so whitespace differences don't false-fail. got must be
// valid JSON. Scrubbers run on got before comparison and on the
// re-marshaled file slice before diffing.
func AssertGoldenJSONField(tb testing.TB, path, field string, got []byte, update bool, scrubbers ...Scrubber) {
	tb.Helper()
	for _, s := range scrubbers {
		got = s(got)
	}

	// Parse got into a generic value so we can both write it back
	// (update path) and re-marshal it canonically (compare path).
	var gotValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		tb.Fatalf("AssertGoldenJSONField: got is not valid JSON: %v", err)
		return
	}

	if update {
		writeJSONField(tb, path, field, gotValue)
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			tb.Fatalf("AssertGoldenJSONField: %s does not exist — run with -update to create it", path)
			return
		}
		tb.Fatalf("AssertGoldenJSONField: read %s: %v", path, err)
		return
	}

	doc := make(map[string]any)
	if uErr := json.Unmarshal(want, &doc); uErr != nil {
		tb.Fatalf("AssertGoldenJSONField: %s is not a JSON object: %v", path, uErr)
		return
	}
	wantValue, ok := doc[field]
	if !ok {
		tb.Fatalf("AssertGoldenJSONField: %s has no field %q — run with -update to add it", path, field)
		return
	}

	wantBytes, err := json.MarshalIndent(wantValue, "", "  ")
	if err != nil {
		tb.Fatalf("AssertGoldenJSONField: re-marshal want: %v", err)
		return
	}
	gotBytes, err := json.MarshalIndent(gotValue, "", "  ")
	if err != nil {
		tb.Fatalf("AssertGoldenJSONField: re-marshal got: %v", err)
		return
	}
	if diff := cmp.Diff(string(wantBytes), string(gotBytes)); diff != "" {
		tb.Fatalf("AssertGoldenJSONField: %s field %q (-want +got)\n%s", path, field, diff)
	}
}

// writeJSONField loads path (or starts with an empty object), sets
// doc[field] = value, and writes the result back canonically. Used
// by the -update path of [AssertGoldenJSONField] so siblings are
// preserved across regenerations.
func writeJSONField(tb testing.TB, path, field string, value any) {
	tb.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		tb.Fatalf("AssertGoldenJSONField: mkdir %s: %v", dir, err)
		return
	}
	doc := make(map[string]any)
	if existing, err := os.ReadFile(path); err == nil {
		if uErr := json.Unmarshal(existing, &doc); uErr != nil {
			tb.Fatalf("AssertGoldenJSONField: %s is not a JSON object: %v", path, uErr)
			return
		}
	}
	doc[field] = value
	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		tb.Fatalf("AssertGoldenJSONField: marshal: %v", err)
		return
	}
	body = append(body, '\n')
	if err := os.WriteFile(path, body, 0o644); err != nil { //nolint:gosec // golden file permissions
		tb.Fatalf("AssertGoldenJSONField: write %s: %v", path, err)
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
