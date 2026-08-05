// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.thesmos.sh/testkit/conformance/gate"
)

// mkCorpus builds a corpus tree under a temporary root. Each entry is a
// directory-relative file path; content is irrelevant to the walker, which
// classifies on names alone.
func mkCorpus(t *testing.T, files ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, f := range files {
		path := filepath.Join(root, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("setup: %v", err)
		}
		if err := os.WriteFile(path, []byte("package p\n"), 0o600); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}
	return root
}

// The walk returns generator *inputs*. A directory holding only emitted files
// is an output, and returning it would have the gate demand fixtures for the
// generated code rather than for the corpus.
func TestWalkSkipsGeneratedOutput(t *testing.T) {
	t.Parallel()

	root := mkCorpus(t,
		"corpus/iface/detector/reader/reader.go",
		"corpus/iface/detector/reader/readertest/reader_stub.gen.go",
	)

	got, err := gate.Walk(root)
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected only the fixture, got %d: %v", len(got), got)
	}
	if got[0].Dir != "corpus/iface/detector/reader" {
		t.Errorf("unexpected fixture dir %q", got[0].Dir)
	}
}

// Test files are not fixtures either: a directory holding only tests has
// nothing to generate from.
func TestWalkSkipsTestOnlyDirectories(t *testing.T) {
	t.Parallel()

	root := mkCorpus(t, "corpus/iface/detector/reader/reader_test.go")

	got, err := gate.Walk(root)
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("a test-only directory is not a fixture, got: %v", got)
	}
}

// The iface tree nests one level deeper than the others for its classification
// axis. Getting the split wrong would file every detector fixture under a
// blank axis and make the report unreadable.
func TestWalkClassifiesPathShape(t *testing.T) {
	t.Parallel()

	root := mkCorpus(t,
		"corpus/iface/detector/reader/reader.go",
		"corpus/iface/mixin/idempotent/idempotent.go",
		"corpus/enum/status/status.go",
		"corpus/errors/store/store.go",
	)

	got, err := gate.Walk(root)
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	want := map[string]gate.Package{
		"corpus/iface/detector/reader": {
			Dir: "corpus/iface/detector/reader", Kind: "iface", Axis: "detector", Name: "reader",
		},
		"corpus/iface/mixin/idempotent": {
			Dir: "corpus/iface/mixin/idempotent", Kind: "iface", Axis: "mixin", Name: "idempotent",
		},
		"corpus/enum/status": {
			Dir: "corpus/enum/status", Kind: "enum", Axis: "", Name: "status",
		},
		"corpus/errors/store": {
			Dir: "corpus/errors/store", Kind: "errors", Axis: "", Name: "store",
		},
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d fixtures, got %d: %v", len(want), len(got), got)
	}
	for _, p := range got {
		exp, ok := want[p.Dir]
		if !ok {
			t.Errorf("unexpected fixture %q", p.Dir)
			continue
		}
		if p != exp {
			t.Errorf("fixture %q: got %+v, want %+v", p.Dir, p, exp)
		}
	}
}

// A gate that reported gaps in filesystem order would produce a different diff
// on every machine, which makes the failure output useless for comparison.
func TestWalkIsOrdered(t *testing.T) {
	t.Parallel()

	root := mkCorpus(t,
		"corpus/iface/detector/writer/writer.go",
		"corpus/iface/detector/reader/reader.go",
		"corpus/enum/status/status.go",
	)

	got, err := gate.Walk(root)
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].Dir > got[i].Dir {
			t.Fatalf("results are unordered at %d: %q then %q", i, got[i-1].Dir, got[i].Dir)
		}
	}
}

func TestWalkImportPath(t *testing.T) {
	t.Parallel()

	p := gate.Package{Dir: "corpus/iface/detector/reader"}
	want := "go.thesmos.sh/testkit/conformance/corpus/iface/detector/reader"

	if got := p.ImportPath(); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

// An absent corpus is a setup error, not an empty result: silently returning
// nothing would let the gate pass on a repository where the corpus failed to
// check out.
func TestWalkReportsMissingCorpus(t *testing.T) {
	t.Parallel()

	if _, err := gate.Walk(t.TempDir()); err == nil {
		t.Fatal("a missing corpus root must be reported, not read as empty")
	}
}

// The corpus root itself can hold a Go file — a doc.go, say — which yields a
// path with no kind segment. That is a partially-populated corpus, a state the
// gate reports rather than one that stops it.
func TestWalkHandlesCorpusRootSource(t *testing.T) {
	t.Parallel()

	root := mkCorpus(t, "corpus/doc.go")

	got, err := gate.Walk(root)
	if err != nil {
		t.Fatalf("a Go file at the corpus root is not an error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected the root itself, got %d: %v", len(got), got)
	}
	if got[0].Kind != "" || got[0].Axis != "" {
		t.Errorf("a path with no kind segment must classify as empty, got %+v", got[0])
	}
}

// A corpus directory the process cannot read is a broken checkout, not an
// empty corpus. Reporting it as empty would let the gate pass on a repository
// where the fixtures never arrived.
//
//nolint:paralleltest // chmod on a shared temp tree
func TestWalkReportsUnreadableDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses permission checks; expires 2027-01-01")
	}

	root := mkCorpus(t, "corpus/iface/detector/reader/reader.go")
	blocked := filepath.Join(root, "corpus", "iface", "detector", "reader")
	if err := os.Chmod(blocked, 0o000); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o750) })

	if _, err := gate.Walk(root); err == nil {
		t.Fatal("an unreadable corpus directory must be reported, not skipped")
	}
}
