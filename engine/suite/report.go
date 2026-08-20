// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ReportFormat versions the machine-readable report. Fields may be added;
// nothing already in it changes meaning. A breaking change gets a new
// version string (ADR-0025).
const ReportFormat = "testkit-report v1"

// Report is what a run produces. It is the source: the text a run logs is
// rendered from this struct, so tooling reads the struct and log wording
// stays free to change.
type Report struct {
	// Format versions the artifact ("testkit-report v1"); a consumer
	// checks it before reading anything else.
	Format string `json:"format"`
	// Suite is the run's display name, [Suite.Name] verbatim.
	Suite string `json:"suite"`

	// ModulePath and PackagePath qualify Suite for fleet-scale scraping:
	// short suite base names collide across a monorepo, and the filename
	// token that disambiguates artifacts is not part of the payload.
	ModulePath  string `json:"module_path,omitempty"`
	PackagePath string `json:"package_path,omitempty"`

	// Subjects and Checks are the headline multiplication: legs is
	// their product minus drops, and every leg must be accounted for.
	Subjects int `json:"subjects"`
	Checks   int `json:"checks"`

	// Oracle names the subject other subjects were compared against, when
	// the run declared one.
	Oracle string `json:"oracle,omitempty"`

	// Legs is one entry per (subject, check) pair that ran.
	Legs []Leg `json:"legs"`

	// Dropped lists the IDs the run was told to skip.
	Dropped []string `json:"dropped,omitempty"`

	// ByClass counts legs per class, filled in when the run ends.
	ByClass map[string]int `json:"byClass,omitempty"`

	// Tiers counts how model checks got their reference, as the checks
	// themselves reported it: "differential" for a declared oracle,
	// "derived" for a reference built from the interface's shape, "twin"
	// for a second copy of the subject. A twin comparison cannot catch a
	// deterministic bug, because both sides have it.
	Tiers map[string]int `json:"tiers,omitempty"`

	// RunFailed is whether the surrounding test failed, recorded when the
	// report is written. It reconciles the report with the test's exit
	// state: an unmet capability's Fatal, a failing cleanup, or a failure
	// outside any leg reddens the run without producing a Failed leg, and
	// a report that said "0 failed" about a red run would be lying.
	RunFailed bool `json:"runFailed"`

	// RapidSeed is the -rapid.seed the property checks ran under, read from
	// the flag when this binary links rapid. "0" means randomized: a
	// failure prints the seed to replay it with. Recorded because a
	// failure nobody can replay is not a finding, and a report that
	// omitted the seed would leave CI unable to say which kind of run it
	// was.
	RapidSeed string `json:"rapidSeed,omitempty"`
}

// Disposition is what became of one leg, and it is the only part of the
// outcome consumers switch on.
//
// Three values, frozen. Everything finer is a [Leg.Reason] in an open
// namespace, and that split is the lesson of adding `vacuous` late: an
// outcome consumers switch on is the worst place in a versioned encoding to
// add a value, and the engine keeps learning new ones — a law whose defect
// class no defect produces, a law no defect reaches, a precondition that
// never engaged. Every one of those is a reason under DidNotRun rather than
// a fourth thing to switch on.
type Disposition string

const (
	// Passed is the leg's verdict when every assertion held.
	Passed Disposition = "passed"
	// Failed is the leg's verdict when any assertion went red.
	Failed Disposition = "failed"

	// DidNotRun covers everything that never reached a verdict. The reason
	// says which — a capability the subject could not meet, preconditions
	// that never engaged, a claim the harness cannot produce a defect for.
	DidNotRun Disposition = "notrun"
)

// The reasons a leg did not run. Open: additions here are additive in the
// encoding, which is the whole point of keeping them out of Disposition.
const (
	// ReasonUnmet is a capability the subject does not provide.
	ReasonUnmet = "unmet"

	// ReasonVacuous is a check whose preconditions never engaged — the
	// poison that did not take, the write that never errored. The engine
	// reports it through its own law census.
	ReasonVacuous = "vacuous"

	// ReasonExcused is a check one subject structurally cannot take. Loud
	// where a skip would be silent — the leg exists and the did-not-run
	// breakdown counts it — and deliberately not "dropped", which is the
	// run-level reviewer decision the Dropped list carries.
	ReasonExcused = "excused"
)

// Leg is one (subject, check) execution: Subject and Check name the
// pair, Class carries the check's report bucket, and Outcome is the
// verdict — Passed, Failed, or DidNotRun narrowed by Reason.
type Leg struct {
	Subject string      `json:"subject"`
	Check   string      `json:"check"`
	Class   string      `json:"class"`
	Outcome Disposition `json:"outcome"`

	// Reason narrows a DidNotRun, empty for a leg that reached a verdict.
	Reason string `json:"reason,omitempty"`

	// Falsifiable is what is known about this check's ability to fail, and
	// Why the argument where it cannot be shown. A green leg whose check
	// nothing has falsified is a different statement from a green leg whose
	// check was driven against a defect and caught it.
	Falsifiable string `json:"falsifiable"`
	// Why carries the argument when a check cannot be shown able to
	// fail — the Argued state's recorded reason.
	Why string `json:"falsifiableWhy,omitempty"`

	// Unengaged lists laws a PASSING leg bound but never engaged: the
	// leg's verdict covers less than its binding, and a reader counting
	// green legs needs to see which greens asserted less than they bound.
	Unengaged []string `json:"unengaged,omitempty"`

	// Tier names how a model leg got its reference (differential,
	// derived, twin); empty for a check that built none — absence is
	// the truth, never a guess.
	Tier string `json:"tier,omitempty"`
}

// legOf is what the runner hands back per leg.
type legOf struct {
	outcome     Disposition
	reason      string
	falsifiable Falsifiability
	tier        string
	unengaged   []string
}

// add records one finished leg. The runner calls it; it takes plain
// values so the report stays free of the subject's type parameter.
func (r *Report) add(subject, check, class string, l legOf) {
	tier := l.tier
	// The zero Falsifiability normalizes to its documented name. Without
	// this, a hand-written check that never called Proven or Argued would
	// encode "" while the constant says "unproven" — two spellings of one
	// state in a versioned format.
	state := l.falsifiable.State
	if state == "" {
		state = FalsifiableUnproven
	}
	r.Legs = append(r.Legs, Leg{
		Subject:     subject,
		Check:       check,
		Class:       class,
		Outcome:     l.outcome,
		Reason:      l.reason,
		Falsifiable: string(state),
		Why:         l.falsifiable.Why,
		Tier:        tier,
		Unengaged:   l.unengaged,
	})
	if tier != "" {
		if r.Tiers == nil {
			r.Tiers = map[string]int{}
		}
		r.Tiers[tier]++
	}
}

func (r *Report) finish() {
	// Only legs that reached a verdict: the by-class line and the "of N
	// checks that ran" sentence must reconcile, and did-not-run legs have
	// their own breakdown.
	r.ByClass = map[string]int{}
	for _, l := range r.Legs {
		if l.Outcome == DidNotRun {
			continue
		}
		r.ByClass[l.Class]++
	}
	sort.Slice(r.Legs, func(i, j int) bool {
		if r.Legs[i].Subject != r.Legs[j].Subject {
			return r.Legs[i].Subject < r.Legs[j].Subject
		}
		return r.Legs[i].Check < r.Legs[j].Check
	})
}

// JSON renders the report in its versioned encoding.
func (r *Report) JSON() ([]byte, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode the report: %w", err)
	}
	return b, nil
}

// Text renders the report for a person reading test output. It is a view
// of the struct and carries no information the struct does not.
func (r *Report) Text() string {
	var b strings.Builder
	passed, failed, notrun := 0, 0, 0
	byReason := map[string]int{}
	for _, l := range r.Legs {
		switch l.Outcome {
		case Passed:
			passed++
		case Failed:
			failed++
		case DidNotRun:
			notrun++
			byReason[l.Reason]++
		}
	}

	fmt.Fprintf(&b, "%s: %d checks x %d subjects = %d legs (%d passed, %d failed, %d did not run)\n",
		r.Suite, r.Checks, r.Subjects, len(r.Legs), passed, failed, notrun)
	if notrun > 0 {
		fmt.Fprintf(&b, "  did not run: %s\n", countList(byReason))
	}
	proven, argued, unproven := r.falsifiableTally()
	fmt.Fprintf(&b, "  %s\n", falsifiableLine(proven, argued, unproven))

	// Naming them rather than only counting them. "2 unproven" tells a
	// reader there is work and not where it is, and the rows in question
	// are always ones somebody hand-wrote — a generated check takes its
	// stamp from the constructor that builds it. Reported, never failed:
	// a claim can be genuinely hard to plant a defect for, and Argued is
	// the answer to that, which is the author's call and not the run's.
	if len(unproven) > 0 {
		fmt.Fprintf(&b, "  unproven: %s — set ProvenBy on each, or Argued with the reason\n",
			strings.Join(unproven, " | "))
	}

	// A green leg that bound five laws and engaged one is not the same
	// green as its neighbors; naming them here is what keeps a passing
	// bundle from reading as full coverage.
	if partial := r.partialLegs(); len(partial) > 0 {
		fmt.Fprintf(&b, "  passed with unengaged laws: %s\n", strings.Join(partial, " | "))
	}

	if len(r.ByClass) > 0 {
		fmt.Fprintf(&b, "  by class: %s\n", countList(r.ByClass))
	}

	if r.RunFailed && failed == 0 {
		b.WriteString("  the test failed without a failed leg: a capability was unmet, " +
			"a cleanup failed, or something failed outside the checks\n")
	}

	if r.Oracle != "" {
		fmt.Fprintf(&b, "  oracle: %q — other subjects compared against it\n", r.Oracle)
	} else if r.Subjects > 1 {
		// Name the tier the run actually fell back to rather than assuming
		// the worst one. A derived reference is a real oracle — weaker than a
		// second implementation, far stronger than a twin — and telling a
		// consumer their run rode the twin floor when it did not is the same
		// species of wrong answer as the reverse.
		fmt.Fprintf(&b, "  oracle: none. %d subjects and no declared reference, so model "+
			"checks fell back to %s. Mark your reference subject .Oracle() to compare "+
			"the implementations against each other.\n", r.Subjects, r.fallbackTier())
	}

	if len(r.Tiers) > 0 {
		fmt.Fprintf(&b, "  model reference: %s\n", countList(r.Tiers))
	}

	if len(r.Dropped) > 0 {
		fmt.Fprintf(&b, "  dropped: %s\n", strings.Join(r.Dropped, ", "))
	}
	switch r.RapidSeed {
	case "":
		// This binary links no property checks; the line would be noise.
	case "0", "randomized":
		b.WriteString("  rapid seed: randomized — a failure prints its replay seed; " +
			"set " + EnvRapidSeed + " to pin presubmit\n")
	default:
		fmt.Fprintf(&b, "  rapid seed: %s (pinned)\n", r.RapidSeed)
	}
	return b.String()
}

// fallbackTier describes what the model legs compared against when no oracle
// was declared, in the words the tier deserves.
func (r *Report) fallbackTier() string {
	switch {
	case r.Tiers[string(TierDerived)] > 0 && r.Tiers[string(TierTwin)] > 0:
		return "a mix of derived references and twins"
	case r.Tiers[string(TierDerived)] > 0:
		return "references derived from the interface's shape, which catch a " +
			"deterministic bug but do not know this implementation's intent"
	case r.Tiers[string(TierTwin)] > 0:
		return "a second copy of each subject, which cannot catch a deterministic " +
			"bug because both copies have it"
	default:
		return "no reference (this run had no model checks)"
	}
}

// falsifiableTally counts the checks that reached a verdict by what is
// known about their ability to fail, and names the unproven ones.
//
// One walk rather than two, so the count and the list cannot disagree.
// A summary saying "3 unproven" above a list of two would be a defect in
// the report, which is the one artifact a reader has no way to check.
func (r *Report) falsifiableTally() (proven, argued int, unproven []string) {
	seen := map[string]bool{}
	for _, l := range r.Legs {
		// Only legs that reached a verdict count. A check whose every leg
		// was unmet, vacuous, or skipped did not run, and counting its
		// Proven stamp here would make the sentence lie about exactly the
		// runs it exists to expose.
		if l.Outcome == DidNotRun {
			continue
		}
		if seen[l.Check] {
			continue // once per check, not once per subject
		}
		seen[l.Check] = true
		switch FalsifiableState(l.Falsifiable) {
		case FalsifiableProven:
			proven++
		case FalsifiableArgued:
			argued++
		default:
			unproven = append(unproven, l.Check)
		}
	}
	return proven, argued, unproven
}

// falsifiableLine is the sentence a conformance statement is written from.
//
// The one a run could not previously make. "44 legs, 44 passed" is true of a
// suite that works and of one that asserts nothing; this says which, and
// where a check cannot be shown able to fail it says so rather than counting
// it among the proven.
func falsifiableLine(proven, argued int, unproven []string) string {
	ran := proven + argued + len(unproven)
	line := fmt.Sprintf("of %d checks that ran: %d proven able to fail", ran, proven)
	if argued > 0 {
		line += fmt.Sprintf(", %d argued", argued)
	}
	if len(unproven) > 0 {
		line += fmt.Sprintf(", %d unproven", len(unproven))
	}
	return line
}

// countList renders a reason histogram in a stable order.
func countList(counts map[string]int) string {
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s %d", k, counts[k]))
	}
	return strings.Join(parts, " | ")
}

// WriteArtifact writes the versioned JSON encoding into dir, named after
// the test and the suite, for CI systems that consume files rather than
// logs. The runner calls it when TESTKIT_REPORT_DIR is set; it is exported
// so a custom runner can emit the same artifact.
func (r *Report) WriteArtifact(dir, testName string) error {
	b, jsonErr := r.JSON()
	if jsonErr != nil {
		return fmt.Errorf("encode the report: %w", jsonErr)
	}
	// The package token disambiguates: two packages sharing a test name
	// and a suite base name would otherwise silently overwrite each other
	// in a shared report directory. Failing to learn it must not degrade
	// to a shared name that reintroduces the collision the token prevents.
	pkg, err := pkgToken()
	if err != nil {
		return err
	}
	name := pkg + "." + artifactName(testName) + "." + artifactName(r.Suite) + ".report.json"
	// The directory is the operator's to name and this run's to ensure:
	// a green suite failing because CI forgot a mkdir is a false red.
	if err := os.MkdirAll(dir, 0o750); err != nil { //nolint:gosec // dir is the operator's TESTKIT_REPORT_DIR
		return fmt.Errorf("create the report directory: %w", err)
	}
	// -count reruns the same test under the same name; first write wins
	// the bare name and every rerun gets an ordinal, so no run destroys
	// another silently.
	name = unusedArtifactName(dir, name)
	// 0o600: a report carries a run's whole assertion inventory and often
	// lands in a shared CI directory.
	//nolint:gosec // G703: TESTKIT_REPORT_DIR is where the operator asked for
	// artifacts; writing under the directory they named is the feature.
	if err := os.WriteFile(filepath.Join(dir, name), b, 0o600); err != nil {
		return fmt.Errorf("write the report artifact: %w", err)
	}
	return nil
}

// pkgToken names the package whose test wrote the artifact: the module
// path plus the module-relative directory, flattened. The base name
// alone was the first answer, and two packages sharing a base name in
// one shared report directory silently overwrote each other; the
// module-relative path alone was the second, and two MODULES sharing a
// relative layout still collided. The module path is what Go already
// guarantees unique. Outside any module the fallback appends a short
// hash of the absolute directory — less readable, still collision-free.
func pkgToken() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve the package directory for the report name: %w", err)
	}
	for dir := wd; ; {
		if gomod, readErr := os.ReadFile(filepath.Join(dir, "go.mod")); readErr == nil {
			if rel, relErr := filepath.Rel(dir, wd); relErr == nil {
				token := modulePath(gomod)
				if token == "" {
					token = filepath.Base(dir)
				}
				if rel != "." {
					token += "/" + filepath.ToSlash(rel)
				}
				return artifactName(token), nil
			}
			// An unrelatable pair takes the hashed fallback below.
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(wd)) // hash.Hash Write never returns an error
	return fmt.Sprintf("%s-%08x", filepath.Base(wd), h.Sum32()), nil
}

// modulePath reads the module declaration from go.mod bytes, empty when
// none parses. Only the unquoted form is read — quoted module paths are
// legal and vanishingly rare, and the caller's fallback (the directory
// base name) stays collision-free enough for a report filename.
func modulePath(gomod []byte) string {
	for line := range strings.SplitSeq(string(gomod), "\n") {
		if path, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.TrimSpace(path)
		}
	}
	return ""
}

// artifactName flattens a test or suite name into one filename segment.
// Subtest names carry slashes, which would scatter artifacts into
// directories nobody created; dots flatten too, because '.' is the
// artifact name's own field separator and a segment carrying its own
// dots would make the fields unparseable.
func artifactName(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ' ', ':', '.':
			return '_'
		default:
			return r
		}
	}, s)
}

// unusedArtifactName returns name, or the first ordinal variant
// (name-2, name-3, …) that does not exist in dir. Existence is the
// only signal available: the testing package does not expose which
// -count attempt is running. dir is the operator's TESTKIT_REPORT_DIR;
// probing under the directory they named is the feature.
//
//nolint:gosec // see above
func unusedArtifactName(dir, name string) string {
	if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
		return name
	}
	base := strings.TrimSuffix(name, ".report.json")
	for i := 2; ; i++ {
		next := fmt.Sprintf("%s-%d.report.json", base, i)
		if _, err := os.Stat(filepath.Join(dir, next)); err != nil {
			return next
		}
	}
}

// partialLegs names the passing legs that bound laws they never
// engaged, each with the laws in question.
func (r *Report) partialLegs() []string {
	var out []string
	for _, l := range r.Legs {
		if l.Outcome == Passed && len(l.Unengaged) > 0 {
			out = append(out, fmt.Sprintf("%s [%s]", l.Check, strings.Join(l.Unengaged, ", ")))
		}
	}
	return out
}
