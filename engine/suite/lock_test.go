// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/suite"
)

func writeFile(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func lockCheck(id suite.ID, claim string) suite.Check[fake] {
	c := check(id)
	c.Claim = claim
	return c
}

func TestRenderLockSortsAcrossSuites(t *testing.T) {
	t.Parallel()

	a := suite.Suite[fake]{Checks: []suite.Check[fake]{lockCheck("B/two", "second")}}
	b := suite.Suite[fake]{Checks: []suite.Check[fake]{lockCheck("A/one", "first")}}

	linesA, err := suite.LockLines(a)
	if err != nil {
		t.Fatalf("LockLines(a): %v", err)
	}
	linesB, err := suite.LockLines(b)
	if err != nil {
		t.Fatalf("LockLines(b): %v", err)
	}
	got := suite.RenderLock(linesA, linesB)
	want := suite.LockFormat + "\nA/one\tsignature/smoke\tfirst\t\nB/two\tsignature/smoke\tsecond\t\n"
	if got != want {
		t.Errorf("RenderLock must merge and sort all suites' rows under one header:\ngot  %q\nwant %q", got, want)
	}
}

func TestLockLinesRefuseUnparseableClaims(t *testing.T) {
	t.Parallel()

	_, err := suite.LockLines(suite.Suite[fake]{
		Checks: []suite.Check[fake]{lockCheck("A/one", "a\tclaim")},
	})
	if err == nil {
		t.Fatal("a claim containing a tab must be refused, not escaped")
	}
	if !strings.Contains(err.Error(), "A/one") {
		t.Errorf("the refusal must name the check, got %v", err)
	}
}

func TestDiffLockReadsAsAnAssertionDiff(t *testing.T) {
	t.Parallel()

	want := "h\nA/one\tc\tkept\nB/two\tc\tdropped\n"
	got := "h\nA/one\tc\tkept\nC/three\tc\tadded\n"
	d := suite.DiffLock(want, got)
	if !strings.Contains(d, "- B/two") || !strings.Contains(d, "+ C/three") {
		t.Errorf("the diff must show the dropped row as - and the new row as +, got:\n%s", d)
	}
	if strings.Contains(d, "A/one") {
		t.Errorf("an unchanged row must not appear in the diff, got:\n%s", d)
	}
}

func TestSuiteIDs(t *testing.T) {
	t.Parallel()

	s := suite.Suite[fake]{Checks: []suite.Check[fake]{check("A/one"), check("B/two")}}.Without("A/one")
	ids := s.IDs()
	if len(ids) != 2 || ids[0] != "A/one" || ids[1] != "B/two" {
		t.Errorf("IDs lists every check in emission order, dropped included: got %v", ids)
	}
}

func TestVerifyIndexReportsBothDirections(t *testing.T) {
	t.Parallel()

	stale := testkit.NewFailableTB()
	suite.VerifyIndex(stale, "s", []suite.ID{"A/one", "A/stale"}, []suite.ID{"A/one"})
	if !stale.Failed() || !strings.Contains(stale.Msg(), "A/stale") {
		t.Errorf("an index entry pointing at nothing must be named: %s", stale.Msg())
	}

	unlisted := testkit.NewFailableTB()
	suite.VerifyIndex(unlisted, "s", []suite.ID{"A/one"}, []suite.ID{"A/one", "A/unlisted"})
	if !unlisted.Failed() || !strings.Contains(unlisted.Msg(), "A/unlisted") {
		t.Errorf("a check the index forgot must be named: %s", unlisted.Msg())
	}

	ok := testkit.NewFailableTB()
	suite.VerifyIndex(ok, "s", []suite.ID{"A/one"}, []suite.ID{"A/one"})
	if ok.Failed() {
		t.Errorf("a matching index must pass: %s", ok.Msg())
	}
}

func TestVerifyDistinctIDs(t *testing.T) {
	t.Parallel()

	f := testkit.NewFailableTB()
	suite.VerifyDistinctIDs(f, []suite.ID{"Get/smoke"}, []suite.ID{"Get/smoke"})
	if !f.Failed() {
		t.Fatal("a cross-interface ID collision must fail")
	}

	ok := testkit.NewFailableTB()
	suite.VerifyDistinctIDs(ok, []suite.ID{"Get/smoke"}, []suite.ID{"Total/smoke"})
	if ok.Failed() {
		t.Errorf("distinct IDs must pass: %s", ok.Msg())
	}
}

// TestVerifyDropHints pins the fallback-detection contract: the string
// form quotes the ID and a typed index path never does, so a missing
// table entry is detected exactly rather than heuristically.
func TestVerifyDropHints(t *testing.T) {
	t.Parallel()

	hint := func(id suite.ID) string {
		if id == "A/typed" {
			return "S.Without(S.Checks.A.Typed())"
		}
		return "S.Without(" + strconv.Quote(string(id)) + ")"
	}

	f := testkit.NewFailableTB()
	suite.VerifyDropHints(f, "s", []suite.ID{"A/typed", "A/missing"}, hint)
	if !f.Failed() || !strings.Contains(f.Msg(), "A/missing") {
		t.Errorf("an ID rendering the string fallback must be named: %s", f.Msg())
	}

	ok := testkit.NewFailableTB()
	suite.VerifyDropHints(ok, "s", []suite.ID{"A/typed"}, hint)
	if ok.Failed() {
		t.Errorf("a fully indexed set must pass: %s", ok.Msg())
	}

	nilHint := testkit.NewFailableTB()
	suite.VerifyDropHints(nilHint, "s", []suite.ID{"A/typed"}, nil)
	if !nilHint.Failed() || !strings.Contains(nilHint.Msg(), "DropHint") {
		t.Errorf("a suite with no renderer must be refused naming the field: %s", nilHint.Msg())
	}
}

func TestVerifyLock(t *testing.T) {
	t.Parallel()

	s := suite.Suite[fake]{Checks: []suite.Check[fake]{lockCheck("A/one", "first")}}
	path := t.TempDir() + "/checks.lock"

	rows, err := suite.LockLines(s)
	if err != nil {
		t.Fatalf("LockLines: %v", err)
	}
	if writeErr := writeFile(path, suite.RenderLock(rows)); writeErr != nil {
		t.Fatalf("stage the manifest: %v", writeErr)
	}
	ok := testkit.NewFailableTB()
	suite.VerifyLock(ok, path, rows)
	if ok.Failed() {
		t.Errorf("a matching manifest must pass: %s", ok.Msg())
	}

	grown := s.With(lockCheck("B/two", "second"))
	grownRows, err := suite.LockLines(grown)
	if err != nil {
		t.Fatalf("LockLines(grown): %v", err)
	}
	f := testkit.NewFailableTB()
	suite.VerifyLock(f, path, grownRows)
	if !f.Failed() {
		t.Fatal("a manifest missing a row must fail")
	}
	msg := strings.Join(f.Logs(), "\n") + f.Msg()
	if !strings.Contains(msg, "+ B/two") {
		t.Errorf("the failure must carry the assertion diff, got: %s", msg)
	}
}

func TestDropHinter(t *testing.T) {
	t.Parallel()

	hint := suite.DropHinter("StoreSuite", map[suite.ID]string{
		"Get/smoke": "StoreSuite.Checks.Get.Smoke()",
	})
	if got := hint("Get/smoke"); got != "StoreSuite.Without(StoreSuite.Checks.Get.Smoke())" {
		t.Errorf("an indexed ID must render its typed path, got %q", got)
	}
	if got := hint("own/hand-written"); got != `StoreSuite.Without("own/hand-written")` {
		t.Errorf("a consumer ID falls back to the string form, which is its truth, got %q", got)
	}
}

//nolint:paralleltest // t.Setenv forbids Parallel.
func TestVerifyLockWriteMode(t *testing.T) {
	lines, err := suite.LockLines(suite.Suite[int]{Checks: []suite.Check[int]{
		{ID: "A/one", Class: suite.ClassSmoke, Claim: "first"},
	}})
	if err != nil {
		t.Fatalf("LockLines: %v", err)
	}
	path := filepath.Join(t.TempDir(), "checks.lock")

	t.Setenv("TESTKIT_LOCK_WRITE", "1")
	suite.VerifyLock(t, path, lines)
	got, err := os.ReadFile(path)
	if err != nil || string(got) != suite.RenderLock(lines) {
		t.Fatalf("write mode must seed the manifest from the emitted set: %v %q", err, got)
	}

	t.Setenv("TESTKIT_LOCK_WRITE", "")
	suite.VerifyLock(t, path, lines) // the seeded manifest must verify byte-for-byte
}

//nolint:paralleltest // shares the env-sensitive VerifyLock path; keep it serial beside the write-mode test.
func TestVerifyLockRefusals(t *testing.T) {
	lines, err := suite.LockLines(suite.Suite[int]{Checks: []suite.Check[int]{
		{ID: "A/one", Class: suite.ClassSmoke, Claim: "first"},
	}})
	if err != nil {
		t.Fatalf("LockLines: %v", err)
	}

	// Fatalf must halt VerifyLock the way it halts a real test, or the
	// missing-manifest case falls through into the comparison below it.
	missing := testkit.NewFailableTB().WithGoexit()
	done := make(chan struct{})
	go func() {
		defer close(done)
		suite.VerifyLock(missing, filepath.Join(t.TempDir(), "absent.lock"), lines)
	}()
	<-done
	if !missing.Failed() || !strings.Contains(missing.Msg(), "no manifest") {
		t.Errorf("a missing manifest must fail with the seeding advice, got %q", missing.Msg())
	}

	path := filepath.Join(t.TempDir(), "checks.lock")
	if err := os.WriteFile(path, []byte("testkit-checks v2\nstale\trow\there\t\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	drifted := testkit.NewFailableTB()
	suite.VerifyLock(drifted, path, lines)
	if !drifted.Failed() || !strings.Contains(drifted.Msg(), "no longer describes") {
		t.Errorf("a drifted manifest must fail with the assertion-diff framing, got %q", drifted.Msg())
	}
}
