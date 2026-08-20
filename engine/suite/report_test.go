// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"go.thesmos.sh/testkit/engine/suite"
)

func TestReportJSONIsVersioned(t *testing.T) {
	t.Parallel()

	r := &suite.Report{Format: suite.ReportFormat, Suite: "s"}
	b, err := r.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if !strings.Contains(string(b), suite.ReportFormat) {
		t.Errorf("the encoding must carry its format version; got %s", b)
	}
}

// TestReportTextNudgesTowardAnOracle pins the message a run gets when it
// has more than one subject and no reference. Without it, model checks
// compare each subject against a copy of itself, which cannot catch a bug
// both copies share.
func TestReportTextNudgesTowardAnOracle(t *testing.T) {
	t.Parallel()

	r := &suite.Report{Format: suite.ReportFormat, Suite: "s", Subjects: 2, Checks: 1}
	text := r.Text()
	if !strings.Contains(text, "Oracle()") {
		t.Errorf("a run with 2 subjects and no oracle must say how to fix it; got:\n%s", text)
	}
}

// TestReportNamesTheUnprovenChecks pins the report line a consumer works
// from: a count alone says there is work without saying where, and the
// rows behind it are always hand-written ones, because a generated check
// takes its stamp from the constructor that builds it.
//
// A report line rather than a failure. A claim can be genuinely hard to
// plant a defect for, and Argued exists for that — which is the author's
// decision, not something a run should force by going red.
func TestReportNamesTheUnprovenChecks(t *testing.T) {
	t.Parallel()

	leg := func(subject, check string, state suite.FalsifiableState, out suite.Disposition) suite.Leg {
		return suite.Leg{
			Subject: subject, Check: check,
			Outcome: out, Falsifiable: string(state),
		}
	}
	r := &suite.Report{Format: suite.ReportFormat, Suite: "s", Subjects: 2, Checks: 4, Legs: []suite.Leg{
		leg("a", "Put/smoke", suite.FalsifiableProven, suite.Passed),
		leg("a", "Put/deadline", suite.FalsifiableArgued, suite.Passed),
		leg("a", "own/newer-value-wins", suite.FalsifiableUnproven, suite.Passed),
		// The same check on a second subject: named once, not once per leg.
		leg("b", "own/newer-value-wins", suite.FalsifiableUnproven, suite.Passed),
		// Unproven and never reached a verdict, so it is not this line's
		// business: the did-not-run breakdown already carries it.
		leg("b", "own/needs-a-clock", suite.FalsifiableUnproven, suite.DidNotRun),
	}}

	text := r.Text()
	if !strings.Contains(text, "of 3 checks that ran: 1 proven able to fail, 1 argued, 1 unproven") {
		t.Errorf("the summary must tally the checks that ran; got:\n%s", text)
	}
	if !strings.Contains(text, "unproven: own/newer-value-wins — set ProvenBy") {
		t.Errorf("the unproven row must be named, with what closes it; got:\n%s", text)
	}
	if strings.Count(text, "own/newer-value-wins") != 1 {
		t.Errorf("a check is named once, not once per subject; got:\n%s", text)
	}
	if strings.Contains(text, "own/needs-a-clock") {
		t.Errorf("a check that never ran is the did-not-run line's, not this one's; got:\n%s", text)
	}
}

// TestReportSaysNothingWhenEveryCheckHasItsEvidence keeps the line from
// becoming noise every green run prints.
func TestReportSaysNothingWhenEveryCheckHasItsEvidence(t *testing.T) {
	t.Parallel()

	r := &suite.Report{Format: suite.ReportFormat, Suite: "s", Subjects: 1, Checks: 1, Legs: []suite.Leg{
		{
			Subject: "a", Check: "Put/smoke", Outcome: suite.Passed,
			Falsifiable: string(suite.FalsifiableProven),
		},
	}}

	text := r.Text()
	if strings.Contains(text, "unproven") {
		t.Errorf("a run with nothing unproven must not mention it; got:\n%s", text)
	}
}

// TestReportArtifact pins the CI egress contract: with TESTKIT_REPORT_DIR
// set, a run writes its versioned JSON beside the log text, named after
// the test and the suite, and the encoding round-trips.
func TestReportArtifact(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(suite.EnvReportDir, dir)

	t.Run("run", func(t *testing.T) {
		s := suite.Suite[fake]{Name: "artifact.Suite", Checks: []suite.Check[fake]{check("A/one")}}
		suite.Run(t, s, suite.Subject[fake]{
			Name: "subject",
			New:  func(testing.TB) fake { return fake{} },
		})
	})

	matches, err := filepath.Glob(filepath.Join(dir, "*.report.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("want exactly one report artifact, got %v (err %v)", matches, err)
	}
	b, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read the artifact: %v", err)
	}
	var rep suite.Report
	if err := json.Unmarshal(b, &rep); err != nil {
		t.Fatalf("the artifact must be the versioned JSON encoding: %v", err)
	}
	if rep.Format != suite.ReportFormat {
		t.Errorf("artifact format %q, want %q", rep.Format, suite.ReportFormat)
	}
	if rep.RunFailed {
		t.Error("a green run's artifact must say so")
	}
	// The suite's own dot flattens: '.' is the artifact name's field
	// separator, and a segment carrying its own would break the fields.
	if !strings.Contains(matches[0], "artifact_Suite") {
		t.Errorf("the artifact name must carry the flattened suite, got %s", matches[0])
	}
}

// TestArtifactPackageToken pins the collision fix: the token is the
// MODULE PATH plus the module-relative directory, flattened — the pair
// Go already guarantees unique — so neither same-base-name packages nor
// same-layout modules can overwrite each other in a shared report
// directory. This test runs in gen/suite under module
// go.thesmos.sh/testkit/gen.
func TestArtifactPackageToken(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := &suite.Report{Format: suite.ReportFormat, Suite: "s"}
	if err := r.WriteArtifact(dir, "TestX"); err != nil {
		t.Fatalf("WriteArtifact: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "*.report.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("want exactly one artifact, got %v (err %v)", matches, err)
	}
	want := "go_thesmos_sh_testkit_engine_suite.TestX.s.report.json"
	if got := filepath.Base(matches[0]); got != want {
		t.Errorf("artifact name %q, want the module-qualified token in %q", got, want)
	}
}

// BenchmarkReportText budgets the renderer. The report prints from
// t.Cleanup on every run, so its cost lands on every suite; the number
// to watch is allocations at a realistic scale — dozens of legs, a
// handful of classes.
func BenchmarkReportText(b *testing.B) {
	r := &suite.Report{Format: suite.ReportFormat, Suite: "bench.Suite", Subjects: 2, Checks: 30}
	for i := range 60 {
		r.Legs = append(r.Legs, suite.Leg{
			Subject:     "subject",
			Check:       "Method/check-" + string(rune('a'+i%26)),
			Class:       "signature/smoke",
			Outcome:     suite.Passed,
			Falsifiable: string(suite.FalsifiableProven),
		})
	}
	r.ByClass = map[string]int{"signature/smoke": 40, "model/laws": 20}
	r.Tiers = map[string]int{"derived": 2}

	b.ReportAllocs()
	for b.Loop() {
		_ = r.Text()
	}
}

// TestArtifactPackageTokenOutsideAModule pins the fallback: with no
// go.mod above the working directory, the token is the base name plus a
// short hash of the absolute path — less readable, still collision-free.
//
//nolint:paralleltest // t.Chdir forbids Parallel; the wd change must not race other tests.
func TestArtifactPackageTokenOutsideAModule(t *testing.T) {
	out := t.TempDir()
	work := t.TempDir()
	t.Chdir(work)

	r := &suite.Report{Format: suite.ReportFormat, Suite: "s"}
	if err := r.WriteArtifact(out, "TestX"); err != nil {
		t.Fatalf("WriteArtifact: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(out, "*.report.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("want exactly one artifact, got %v (err %v)", matches, err)
	}
	pattern := "^" + regexp.QuoteMeta(filepath.Base(work)) + `-[0-9a-f]{8}\.TestX\.s\.report\.json$`
	if base := filepath.Base(matches[0]); !regexp.MustCompile(pattern).MatchString(base) {
		t.Errorf("artifact name %q, want the hashed fallback matching %q", base, pattern)
	}
}

func TestWriteArtifactCreatesAndNeverOverwrites(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "nested", "reports")

	rep := &suite.Report{Format: suite.ReportFormat, Suite: "S", Subjects: 1, Checks: 1}
	if err := rep.WriteArtifact(dir, "TestX"); err != nil {
		t.Fatalf("a missing directory is the run's to create: %v", err)
	}
	if err := rep.WriteArtifact(dir, "TestX"); err != nil {
		t.Fatalf("a -count rerun must not fail: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) != 2 {
		t.Fatalf("the rerun must get an ordinal, not destroy run one: %v", names)
	}
	joined := strings.Join(names, " ")
	if !strings.Contains(joined, ".S.report.json") || !strings.Contains(joined, ".S-2.report.json") {
		t.Errorf("first write wins the bare name, the rerun takes the first free ordinal: %v", names)
	}
}
