// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tests use the internal assertGolden with explicit update parameter so we can
// exercise both paths without the -update flag. Each subtest uses t.Chdir to
// an isolated temp directory.

func TestAssertGolden(t *testing.T) { //nolint:paralleltest // golden file tests mutate working directory
	t.Run("missing file without update fails", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		f := NewFailableTB()
		assertGolden(f, "missing.txt", []byte("data"), false)
		True(t, f.Failed(), "must fail when golden file missing")
		True(t, strings.Contains(f.Msg(), "-update"), "must cite -update flag")
	})

	t.Run("update creates file", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		f := NewFailableTB()
		assertGolden(f, "created.txt", []byte("new content"), true)
		False(t, f.Failed(), "update mode must not fail")

		data, err := os.ReadFile(filepath.Join("testdata", "golden", "created.txt"))
		NoError(t, err, "file must exist after update")
		Equal(t, string(data), "new content", "file must have correct content")
	})

	t.Run("update overwrites existing file", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		writeGolden(t, "overwrite.txt", []byte("old"))

		f := NewFailableTB()
		assertGolden(f, "overwrite.txt", []byte("new"), true)
		False(t, f.Failed(), "update must not fail")

		data, err := os.ReadFile(filepath.Join("testdata", "golden", "overwrite.txt"))
		NoError(t, err, "must read overwritten file")
		Equal(t, string(data), "new", "must contain new content")
	})

	t.Run("matching content passes", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		writeGolden(t, "match.txt", []byte("hello"))

		f := NewFailableTB()
		assertGolden(f, "match.txt", []byte("hello"), false)
		False(t, f.Failed(), "must pass when content matches")
	})

	t.Run("different content fails with diff", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		writeGolden(t, "diff.txt", []byte("want"))

		f := NewFailableTB()
		assertGolden(f, "diff.txt", []byte("got"), false)
		True(t, f.Failed(), "must fail when content differs")
		True(t, strings.Contains(f.Msg(), "-want +got"), "must include cmp.Diff markers")
	})

	t.Run("scrubbers applied before comparison", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		writeGolden(t, "scrub.txt", []byte("id=SCRUBBED_RUN"))

		f := NewFailableTB()
		assertGolden(f, "scrub.txt", []byte("id=run_abcdef0123456789"), false, ScrubRunIDs())
		False(t, f.Failed(), "scrubbed content must match golden")
	})

	t.Run("update with scrubbers writes scrubbed content", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		f := NewFailableTB()
		assertGolden(f, "scrub-update.txt", []byte("ts=2026-05-02T12:00:00Z"), true, ScrubTimestamps())
		False(t, f.Failed(), "update must not fail")

		data, err := os.ReadFile(filepath.Join("testdata", "golden", "scrub-update.txt"))
		NoError(t, err, "file must exist")
		Equal(t, string(data), "ts=SCRUBBED_TS", "must write scrubbed content")
	})

	t.Run("public AssertGolden delegates to assertGolden", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		writeGolden(t, "public.txt", []byte("ok"))

		f := NewFailableTB()
		AssertGolden(f, "public.txt", []byte("ok"))
		False(t, f.Failed(), "public API must delegate correctly")
	})

	t.Run("read error on directory-as-file fails", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		goldenDir := filepath.Join("testdata", "golden")
		err := os.MkdirAll(filepath.Join(goldenDir, "isdir"), 0o750)
		if err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}

		f := NewFailableTB()
		assertGolden(f, "isdir", []byte("data"), false)
		True(t, f.Failed(), "must fail on read error")
		True(t, strings.Contains(f.Msg(), "read"), "must mention read error")
	})
}

func TestShouldUpdate(t *testing.T) {
	t.Parallel()

	t.Run("returns false by default", func(t *testing.T) {
		t.Parallel()
		False(t, ShouldUpdate(), "default must be false")
	})
}

func TestScrubTimestamps(t *testing.T) {
	t.Parallel()

	t.Run("replaces ISO-8601 timestamp", func(t *testing.T) {
		t.Parallel()
		s := ScrubTimestamps()
		got := string(s([]byte(`{"created":"2026-05-02T13:45:00Z"}`)))
		True(t, strings.Contains(got, "SCRUBBED_TS"), "must replace timestamp")
		False(t, strings.Contains(got, "2026"), "must remove year")
	})
}

func TestScrubHashes(t *testing.T) {
	t.Parallel()

	t.Run("replaces hex digest", func(t *testing.T) {
		t.Parallel()
		s := ScrubHashes()
		hash := strings.Repeat("ab", 20)
		got := string(s([]byte(`hash: ` + hash)))
		True(t, strings.Contains(got, "SCRUBBED_HASH"), "must replace hash")
	})
}

func TestScrubRunIDs(t *testing.T) {
	t.Parallel()

	t.Run("replaces run ID", func(t *testing.T) {
		t.Parallel()
		s := ScrubRunIDs()
		got := string(s([]byte(`run_abcdef0123456789`)))
		Equal(t, got, "SCRUBBED_RUN", "must replace run ID")
	})
}

func TestScrubJSONFields(t *testing.T) {
	t.Parallel()

	t.Run("replaces named field values", func(t *testing.T) {
		t.Parallel()
		s := ScrubJSONFields("token", "created_at")
		input := `{"token":"secret123","created_at":"2026-05-02","name":"alice"}`
		got := string(s([]byte(input)))
		True(t, strings.Contains(got, `"token":"SCRUBBED"`), "must scrub token")
		True(t, strings.Contains(got, `"name":"alice"`), "must preserve other fields")
	})

	t.Run("replaces numeric values", func(t *testing.T) {
		t.Parallel()
		s := ScrubJSONFields("count")
		got := string(s([]byte(`{"count": 42}`)))
		True(t, strings.Contains(got, "SCRUBBED"), "must scrub numeric value")
	})
}

// writeGolden creates a golden file in the current directory's testdata/golden/.
func writeGolden(t *testing.T, name string, content []byte) {
	t.Helper()
	goldenDir := filepath.Join("testdata", "golden")
	err := os.MkdirAll(goldenDir, 0o750)
	if err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	err = os.WriteFile(filepath.Join(goldenDir, name), content, 0o644) //nolint:gosec
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
