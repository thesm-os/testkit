// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package golden_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/golden"
)

func TestAssertGolden(t *testing.T) { //nolint:paralleltest // golden file tests mutate working directory
	t.Run("missing file without update fails", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		f := testkit.NewFailableTB()
		golden.AssertGolden(f, "missing.txt", []byte("data"), false)
		testkit.True(t, f.Failed(), "must fail when golden file missing")
		testkit.True(t, strings.Contains(f.Msg(), "-update"), "must cite -update flag")
	})

	t.Run("update creates file", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		f := testkit.NewFailableTB()
		golden.AssertGolden(f, "created.txt", []byte("new content"), true)
		testkit.False(t, f.Failed(), "update mode must not fail")

		data, err := os.ReadFile(filepath.Join("testdata", "golden", "created.txt"))
		testkit.NoError(t, err, "file must exist after update")
		testkit.Equal(t, string(data), "new content", "file must have correct content")
	})

	t.Run("update overwrites existing file", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		writeGolden(t, "overwrite.txt", []byte("old"))

		f := testkit.NewFailableTB()
		golden.AssertGolden(f, "overwrite.txt", []byte("new"), true)
		testkit.False(t, f.Failed(), "update must not fail")

		data, err := os.ReadFile(filepath.Join("testdata", "golden", "overwrite.txt"))
		testkit.NoError(t, err, "must read overwritten file")
		testkit.Equal(t, string(data), "new", "must contain new content")
	})

	t.Run("matching content passes", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		writeGolden(t, "match.txt", []byte("hello"))

		f := testkit.NewFailableTB()
		golden.AssertGolden(f, "match.txt", []byte("hello"), false)
		testkit.False(t, f.Failed(), "must pass when content matches")
	})

	t.Run("different content fails with diff", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		writeGolden(t, "diff.txt", []byte("want"))

		f := testkit.NewFailableTB()
		golden.AssertGolden(f, "diff.txt", []byte("got"), false)
		testkit.True(t, f.Failed(), "must fail when content differs")
		testkit.True(t, strings.Contains(f.Msg(), "-want +got"), "must include cmp.Diff markers")
	})

	t.Run("scrubbers applied before comparison", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		writeGolden(t, "scrub.txt", []byte("id=SCRUBBED_RUN"))

		f := testkit.NewFailableTB()
		golden.AssertGolden(f, "scrub.txt", []byte("id=run_abcdef0123456789"), false, golden.ScrubRunIDs())
		testkit.False(t, f.Failed(), "scrubbed content must match golden")
	})

	t.Run("update with scrubbers writes scrubbed content", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		f := testkit.NewFailableTB()
		golden.AssertGolden(f, "scrub-update.txt", []byte("ts=2026-05-02T12:00:00Z"), true, golden.ScrubTimestamps())
		testkit.False(t, f.Failed(), "update must not fail")

		data, err := os.ReadFile(filepath.Join("testdata", "golden", "scrub-update.txt"))
		testkit.NoError(t, err, "file must exist")
		testkit.Equal(t, string(data), "ts=SCRUBBED_TS", "must write scrubbed content")
	})

	t.Run("read error on directory-as-file fails", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		goldenDir := filepath.Join("testdata", "golden")
		err := os.MkdirAll(filepath.Join(goldenDir, "isdir"), 0o750)
		if err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}

		f := testkit.NewFailableTB()
		golden.AssertGolden(f, "isdir", []byte("data"), false)
		testkit.True(t, f.Failed(), "must fail on read error")
		testkit.True(t, strings.Contains(f.Msg(), "read"), "must mention read error")
	})
}

func TestAssertGoldenAt(t *testing.T) { //nolint:paralleltest // golden file tests mutate working directory
	t.Run("path argument is taken verbatim (no testdata/golden prefix)", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		f := testkit.NewFailableTB()
		// Update mode writes to the literal path given.
		golden.AssertGoldenAt(f, "fixtures/sub/file.txt", []byte("hello"), true)
		testkit.False(t, f.Failed(), "update must not fail")

		data, err := os.ReadFile(filepath.Join("fixtures", "sub", "file.txt"))
		testkit.NoError(t, err, "file at literal path")
		testkit.Equal(t, string(data), "hello", "content")
	})

	t.Run("missing file without update fails and cites path", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		f := testkit.NewFailableTB()
		golden.AssertGoldenAt(f, "custom/path.json", []byte("data"), false)
		testkit.True(t, f.Failed(), "missing file must fail")
		testkit.True(t, strings.Contains(f.Msg(), "custom/path.json"), "must cite the literal path")
		testkit.True(t, strings.Contains(f.Msg(), "-update"), "must cite -update flag")
	})

	t.Run("matching content passes", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		dir := filepath.Join("any", "where")
		err := os.MkdirAll(dir, 0o750)
		if err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		err = os.WriteFile(filepath.Join(dir, "g.txt"), []byte("ok"), 0o644) //nolint:gosec
		if err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		f := testkit.NewFailableTB()
		golden.AssertGoldenAt(f, filepath.Join(dir, "g.txt"), []byte("ok"), false)
		testkit.False(t, f.Failed(), "matching content")
	})

	t.Run("AssertGolden delegates to AssertGoldenAt under testdata/golden", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		f := testkit.NewFailableTB()
		golden.AssertGolden(f, "delegated.txt", []byte("body"), true)
		testkit.False(t, f.Failed(), "delegate update")

		data, err := os.ReadFile(filepath.Join("testdata", "golden", "delegated.txt"))
		testkit.NoError(t, err, "convention path")
		testkit.Equal(t, string(data), "body", "content")
	})
}

func TestAssertGoldenJSONField(t *testing.T) { //nolint:paralleltest // mutates working directory
	t.Run("update creates a new file with one field", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		f := testkit.NewFailableTB()
		got := []byte(`{"A": 1, "B": 2}`)
		golden.AssertGoldenJSONField(f, "wire.json", "Status", got, true)
		testkit.False(t, f.Failed(), "update must not fail")

		body, err := os.ReadFile("wire.json")
		testkit.NoError(t, err, "file written")
		testkit.True(t, strings.Contains(string(body), `"Status"`), "field present")
		testkit.True(t, strings.Contains(string(body), `"A": 1`), "value present")
	})

	t.Run("update preserves sibling fields", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		// Seed with two fields.
		f := testkit.NewFailableTB()
		golden.AssertGoldenJSONField(f, "wire.json", "Status",
			[]byte(`{"StatusA": 1}`), true)
		golden.AssertGoldenJSONField(f, "wire.json", "Priority",
			[]byte(`{"PriorityX": 10}`), true)
		testkit.False(t, f.Failed(), "two updates")

		// Update only Status; Priority must remain.
		golden.AssertGoldenJSONField(f, "wire.json", "Status",
			[]byte(`{"StatusA": 1, "StatusB": 2}`), true)
		testkit.False(t, f.Failed(), "third update")

		body, _ := os.ReadFile("wire.json")
		testkit.True(t, strings.Contains(string(body), `"Status"`), "Status present")
		testkit.True(t, strings.Contains(string(body), `"StatusB": 2`), "new value present")
		testkit.True(t, strings.Contains(string(body), `"Priority"`), "Priority preserved")
		testkit.True(t, strings.Contains(string(body), `"PriorityX": 10`), "Priority value preserved")
	})

	t.Run("missing file without update fails", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		f := testkit.NewFailableTB()
		golden.AssertGoldenJSONField(f, "missing.json", "Foo",
			[]byte(`{"x": 1}`), false)
		testkit.True(t, f.Failed(), "missing file must fail")
		testkit.True(t, strings.Contains(f.Msg(), "-update"), "cites flag")
	})

	t.Run("missing field fails with field name in diagnostic", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		err := os.WriteFile("wire.json", []byte(`{"OtherType": {}}`), 0o644) //nolint:gosec
		testkit.NoError(t, err, "seed file")

		f := testkit.NewFailableTB()
		golden.AssertGoldenJSONField(f, "wire.json", "Missing",
			[]byte(`{"x": 1}`), false)
		testkit.True(t, f.Failed(), "missing field fails")
		testkit.True(t, strings.Contains(f.Msg(), `"Missing"`), "names the field")
	})

	t.Run("matching field passes", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		f := testkit.NewFailableTB()
		golden.AssertGoldenJSONField(f, "wire.json", "Status",
			[]byte(`{"A": 1}`), true)
		testkit.False(t, f.Failed(), "seed")

		// Same content compared.
		f2 := testkit.NewFailableTB()
		golden.AssertGoldenJSONField(f2, "wire.json", "Status",
			[]byte(`{"A": 1}`), false)
		testkit.False(t, f2.Failed(), "matching content")
	})

	t.Run("differing field fails with cmp.Diff scoped to the slice", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		f := testkit.NewFailableTB()
		golden.AssertGoldenJSONField(f, "wire.json", "Status",
			[]byte(`{"A": 1}`), true)
		testkit.False(t, f.Failed(), "seed")

		f2 := testkit.NewFailableTB()
		golden.AssertGoldenJSONField(f2, "wire.json", "Status",
			[]byte(`{"A": 2}`), false)
		testkit.True(t, f2.Failed(), "differing content fails")
		testkit.True(t, strings.Contains(f2.Msg(), `"Status"`), "names the field")
		testkit.True(t, strings.Contains(f2.Msg(), "-want +got"), "cmp.Diff markers")
	})

	t.Run("structural comparison ignores whitespace differences", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		// Seed with verbose formatting.
		err := os.WriteFile("wire.json", []byte(`{"Status": {  "A":  1  }}`), 0o644) //nolint:gosec
		testkit.NoError(t, err, "seed")

		f := testkit.NewFailableTB()
		golden.AssertGoldenJSONField(f, "wire.json", "Status",
			[]byte(`{"A":1}`), false)
		testkit.False(t, f.Failed(), "compact got matches verbose stored")
	})

	t.Run("invalid got JSON fails fast", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		f := testkit.NewFailableTB()
		golden.AssertGoldenJSONField(f, "wire.json", "Status",
			[]byte(`not json`), true)
		testkit.True(t, f.Failed(), "invalid got rejected up front")
	})

	t.Run("non-object file body fails with diagnostic", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		// File exists but is a JSON array — can't extract a field.
		err := os.WriteFile("wire.json", []byte(`[1, 2, 3]`), 0o644) //nolint:gosec
		testkit.NoError(t, err, "seed array file")

		f := testkit.NewFailableTB()
		golden.AssertGoldenJSONField(f, "wire.json", "Status",
			[]byte(`{"A": 1}`), false)
		testkit.True(t, f.Failed(), "non-object file fails")
		testkit.True(t, strings.Contains(f.Msg(), "JSON object"),
			"diagnostic explains the shape constraint")
	})

	t.Run("scrubbers applied to got before comparison", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		f := testkit.NewFailableTB()
		golden.AssertGoldenJSONField(f, "wire.json", "Status",
			[]byte(`{"ts": "SCRUBBED_TS"}`), true)
		testkit.False(t, f.Failed(), "seed")

		f2 := testkit.NewFailableTB()
		golden.AssertGoldenJSONField(f2, "wire.json", "Status",
			[]byte(`{"ts": "2026-05-09T10:00:00Z"}`), false, golden.ScrubTimestamps())
		testkit.False(t, f2.Failed(), "scrubbed got must match stored golden")
	})

	t.Run("read error on directory-as-file fails", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		err := os.MkdirAll("wire.json/sub", 0o750)
		testkit.NoError(t, err, "seed dir")

		f := testkit.NewFailableTB()
		golden.AssertGoldenJSONField(f, "wire.json", "Status",
			[]byte(`{"A": 1}`), false)
		testkit.True(t, f.Failed(), "read error must fail")
		testkit.True(t, strings.Contains(f.Msg(), "read"), "must mention read error")
	})

	t.Run("update of non-object file body fails", func(t *testing.T) { //nolint:paralleltest
		t.Chdir(t.TempDir())
		err := os.WriteFile("wire.json", []byte(`[1, 2, 3]`), 0o644) //nolint:gosec
		testkit.NoError(t, err, "seed array file")

		f := testkit.NewFailableTB()
		golden.AssertGoldenJSONField(f, "wire.json", "Status",
			[]byte(`{"A": 1}`), true)
		testkit.True(t, f.Failed(), "update on non-object body fails")
	})
}

func TestCompare(t *testing.T) {
	t.Parallel()

	t.Run("equal content returns empty diff", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, golden.Compare([]byte("a"), []byte("a")), "",
			"matching content has no diff")
	})

	t.Run("different content returns cmp.Diff with markers", func(t *testing.T) {
		t.Parallel()
		diff := golden.Compare([]byte("want"), []byte("got"))
		testkit.True(t, diff != "", "differing content yields a diff")
		testkit.True(t, strings.Contains(diff, "-"), "diff has - marker for want")
		testkit.True(t, strings.Contains(diff, "+"), "diff has + marker for got")
	})

	t.Run("scrubbers run on got before comparison", func(t *testing.T) {
		t.Parallel()
		// Without scrubber, content differs.
		testkit.True(t,
			golden.Compare([]byte("id=SCRUBBED_RUN"), []byte("id=run_abcdef0123456789")) != "",
			"unscrubbed differs")
		// With scrubber, content matches.
		testkit.Equal(t,
			golden.Compare([]byte("id=SCRUBBED_RUN"), []byte("id=run_abcdef0123456789"),
				golden.ScrubRunIDs()),
			"",
			"scrubber reconciles got with want")
	})
}

func TestShouldUpdate(t *testing.T) {
	t.Parallel()

	t.Run("returns false by default", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, golden.ShouldUpdate(), "default must be false")
	})
}

func TestScrubTimestamps(t *testing.T) {
	t.Parallel()

	t.Run("replaces ISO-8601 timestamp", func(t *testing.T) {
		t.Parallel()
		s := golden.ScrubTimestamps()
		got := string(s([]byte(`{"created":"2026-05-02T13:45:00Z"}`)))
		testkit.True(t, strings.Contains(got, "SCRUBBED_TS"), "must replace timestamp")
		testkit.False(t, strings.Contains(got, "2026"), "must remove year")
	})
}

func TestScrubHashes(t *testing.T) {
	t.Parallel()

	t.Run("replaces hex digest", func(t *testing.T) {
		t.Parallel()
		s := golden.ScrubHashes()
		hash := strings.Repeat("ab", 20)
		got := string(s([]byte(`hash: ` + hash)))
		testkit.True(t, strings.Contains(got, "SCRUBBED_HASH"), "must replace hash")
	})
}

func TestScrubRunIDs(t *testing.T) {
	t.Parallel()

	t.Run("replaces run ID", func(t *testing.T) {
		t.Parallel()
		s := golden.ScrubRunIDs()
		got := string(s([]byte(`run_abcdef0123456789`)))
		testkit.Equal(t, got, "SCRUBBED_RUN", "must replace run ID")
	})
}

func TestScrubJSONFields(t *testing.T) {
	t.Parallel()

	t.Run("replaces named field values", func(t *testing.T) {
		t.Parallel()
		s := golden.ScrubJSONFields("token", "created_at")
		input := `{"token":"secret123","created_at":"2026-05-02","name":"alice"}`
		got := string(s([]byte(input)))
		testkit.True(t, strings.Contains(got, `"token":"SCRUBBED"`), "must scrub token")
		testkit.True(t, strings.Contains(got, `"name":"alice"`), "must preserve other fields")
	})

	t.Run("replaces numeric values", func(t *testing.T) {
		t.Parallel()
		s := golden.ScrubJSONFields("count")
		got := string(s([]byte(`{"count": 42}`)))
		testkit.True(t, strings.Contains(got, "SCRUBBED"), "must scrub numeric value")
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

// The filesystem error paths are the ones a passing run never touches, and
// they are exactly where a silent failure would be worst — a golden that
// cannot be written must fail loudly, not appear to update.
func TestGoldenFilesystemFailures(t *testing.T) { //nolint:paralleltest // golden file tests touch the filesystem
	// A regular file where a directory needs to be: MkdirAll and WriteFile
	// both fail against it.
	blockedDir := func(t *testing.T) string {
		t.Helper()
		blocker := filepath.Join(t.TempDir(), "blocker")
		if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
			t.Fatalf("setup: %v", err)
		}
		return filepath.Join(blocker, "sub", "golden.txt")
	}

	t.Run("AssertGoldenAt reports an unmakeable directory", func(t *testing.T) { //nolint:paralleltest // filesystem
		f := testkit.NewFailableTB()
		golden.AssertGoldenAt(f, blockedDir(t), []byte("x"), true)
		if !f.Failed() {
			t.Fatal("an un-creatable directory must fail the assertion")
		}
		if !strings.Contains(f.Msg(), "mkdir") {
			t.Fatalf("the diagnostic must name the failed step, got: %s", f.Msg())
		}
	})

	t.Run("AssertGoldenAt reports an unwritable file", func(t *testing.T) { //nolint:paralleltest // filesystem
		dir := t.TempDir()
		path := filepath.Join(dir, "sub")
		if err := os.MkdirAll(path, 0o750); err != nil {
			t.Fatalf("setup: %v", err)
		}
		// path is a directory, so WriteFile against it fails.
		f := testkit.NewFailableTB()
		golden.AssertGoldenAt(f, path, []byte("x"), true)
		if !f.Failed() {
			t.Fatal("writing over a directory must fail the assertion")
		}
	})

	//nolint:paralleltest // filesystem
	t.Run("AssertGoldenAt distinguishes missing from unreadable", func(t *testing.T) {
		f := testkit.NewFailableTB()
		golden.AssertGoldenAt(f, filepath.Join(t.TempDir(), "absent.txt"), []byte("x"), false)
		if !f.Failed() {
			t.Fatal("a missing golden must fail")
		}
		if !strings.Contains(f.Msg(), "-update") {
			t.Fatalf("a missing golden must tell the reader how to create it, got: %s", f.Msg())
		}

		// A directory read as a file is present but unreadable, which is a
		// different diagnostic.
		dir := t.TempDir()
		f2 := testkit.NewFailableTB()
		golden.AssertGoldenAt(f2, dir, []byte("x"), false)
		if !f2.Failed() {
			t.Fatal("an unreadable golden must fail")
		}
		if strings.Contains(f2.Msg(), "-update") {
			t.Fatalf("an unreadable golden is not a missing one, got: %s", f2.Msg())
		}
	})

	//nolint:paralleltest // filesystem
	t.Run(
		"AssertGoldenJSONField reports an unmakeable directory",
		func(t *testing.T) {
			f := testkit.NewFailableTB()
			golden.AssertGoldenJSONField(f, blockedDir(t), "field", []byte(`{"a":1}`), true)
			if !f.Failed() {
				t.Fatal("an un-creatable directory must fail the assertion")
			}
		},
	)

	t.Run("AssertGoldenJSONField rejects a non-object golden", func(t *testing.T) { //nolint:paralleltest // filesystem
		path := filepath.Join(t.TempDir(), "g.json")
		if err := os.WriteFile(path, []byte(`["not an object"]`), 0o600); err != nil {
			t.Fatalf("setup: %v", err)
		}
		f := testkit.NewFailableTB()
		golden.AssertGoldenJSONField(f, path, "field", []byte(`{"a":1}`), true)
		if !f.Failed() {
			t.Fatal("a golden that is not a JSON object must fail")
		}
	})

	// Updating one field must not disturb its siblings — that is the whole
	// reason writeJSONField reads before it writes.
	t.Run("AssertGoldenJSONField preserves sibling fields", func(t *testing.T) { //nolint:paralleltest // filesystem
		path := filepath.Join(t.TempDir(), "g.json")
		if err := os.WriteFile(path, []byte(`{"keep":"me"}`), 0o600); err != nil {
			t.Fatalf("setup: %v", err)
		}
		f := testkit.NewFailableTB()
		golden.AssertGoldenJSONField(f, path, "added", []byte(`{"a":1}`), true)
		if f.Failed() {
			t.Fatalf("the update must succeed: %s", f.Msg())
		}
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read back: %v", err)
		}
		if !strings.Contains(string(body), `"keep"`) {
			t.Fatalf("sibling fields must survive an update, got: %s", body)
		}
		if !strings.Contains(string(body), `"added"`) {
			t.Fatalf("the updated field must be present, got: %s", body)
		}
	})

	t.Run("AssertGoldenJSONField reports a missing field", func(t *testing.T) { //nolint:paralleltest // filesystem
		path := filepath.Join(t.TempDir(), "g.json")
		if err := os.WriteFile(path, []byte(`{"other":1}`), 0o600); err != nil {
			t.Fatalf("setup: %v", err)
		}
		f := testkit.NewFailableTB()
		golden.AssertGoldenJSONField(f, path, "absent", []byte(`{"a":1}`), false)
		if !f.Failed() {
			t.Fatal("a golden missing the requested field must fail")
		}
		if !strings.Contains(f.Msg(), "-update") {
			t.Fatalf("the diagnostic must say how to add it, got: %s", f.Msg())
		}
	})

	// mkdir succeeds when the parent already exists, so an existing
	// *directory* at the golden path is the only way past it to the write.
	t.Run("AssertGoldenJSONField reports an unwritable golden", func(t *testing.T) { //nolint:paralleltest // filesystem
		path := filepath.Join(t.TempDir(), "g.json")
		if err := os.MkdirAll(path, 0o750); err != nil {
			t.Fatalf("setup: %v", err)
		}
		f := testkit.NewFailableTB()
		golden.AssertGoldenJSONField(f, path, "field", []byte(`{"a":1}`), true)
		if !f.Failed() {
			t.Fatal("a golden path that cannot be written must fail the assertion")
		}
		if !strings.Contains(f.Msg(), "write") {
			t.Fatalf("the diagnostic must name the failed step, got: %s", f.Msg())
		}
	})
}
