// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// LockFormat versions the checks.lock manifest. Rows may be added; a row's
// meaning never changes. A breaking change gets a new version string.
const LockFormat = "testkit-checks v2"

// LockLines renders one suite's manifest rows: ID, class, claim,
// tab-separated. The library owns the format, so every generated package
// and every tool renders it the same way instead of each re-embedding it.
//
// A claim carrying a tab or a newline would make its row unparseable, and
// the format's answer is refusal rather than escaping: claims are authored
// prose, so the author can simply not write one. The refusal is an error
// rather than a panic — this is exported for tooling that is not a test
// binary, and a renderer that kills its caller over input data is not one
// such a tool can use.
func LockLines[S any](s Suite[S]) ([]string, error) {
	lines := make([]string, 0, len(s.Checks))
	for _, c := range s.Checks {
		if strings.ContainsAny(string(c.Class), "\t\n") {
			return nil, fmt.Errorf(
				"check %q has a class containing a tab or newline, which the manifest "+
					"format refuses: %q", c.ID, c.Class,
			)
		}
		if strings.ContainsAny(c.Claim, "\t\n") {
			return nil, fmt.Errorf(
				"check %q has a claim containing a tab or newline, which the manifest "+
					"format refuses: %q", c.ID, c.Claim,
			)
		}
		for _, bind := range c.Binds {
			if strings.ContainsAny(bind, "\t\n,") {
				return nil, fmt.Errorf(
					"check %q has a binding containing a tab, newline or comma, which "+
						"the manifest format refuses: %q", c.ID, bind,
				)
			}
		}
		lines = append(lines, fmt.Sprintf("%s\t%s\t%s\t%s", c.ID, c.Class, c.Claim, strings.Join(c.Binds, ",")))
	}
	return lines, nil
}

// RenderLock composes the manifest from every suite's rows. One generated
// package holds one lock: the ID space is unique per package, so all its
// interfaces' rows land in one sorted file rather than in one file each.
func RenderLock(groups ...[]string) string {
	var all []string
	for _, g := range groups {
		all = append(all, g...)
	}
	sort.Strings(all)

	var b strings.Builder
	b.WriteString(LockFormat)
	b.WriteString("\n")
	for _, l := range all {
		b.WriteString(l)
		b.WriteString("\n")
	}
	return b.String()
}

// DiffLock renders the rows removed from and added to a manifest, which is
// the whole point of the file: a reviewer reads assertions, not generated
// code. A removed row is a weakened assertion set and deserves exactly the
// attention a red minus sign draws.
func DiffLock(want, got string) string {
	inWant := map[string]bool{}
	for l := range strings.SplitSeq(want, "\n") {
		inWant[l] = true
	}
	inGot := map[string]bool{}
	for l := range strings.SplitSeq(got, "\n") {
		inGot[l] = true
	}

	var b strings.Builder
	for l := range strings.SplitSeq(want, "\n") {
		if l != "" && !inGot[l] {
			fmt.Fprintf(&b, "  - %s\n", l)
		}
	}
	for l := range strings.SplitSeq(got, "\n") {
		if l != "" && !inWant[l] {
			fmt.Fprintf(&b, "  + %s\n", l)
		}
	}
	return b.String()
}

// --- Self-check verifiers --------------------------------------------------
//
// The invariants every generated package owes about itself, as library
// functions: the generated self-check test is enumeration only — which
// suites, which index, which file — because only the package knows that,
// and nothing else about verifying it is package-specific. A consumer
// writes none of this.

// VerifyLock fails when the manifest on disk no longer describes the
// checks the given suites emit. The diff reads as the assertion diff it
// is: a removed row is a weakened assertion set.
func VerifyLock(tb testing.TB, path string, groups ...[]string) {
	tb.Helper()
	got := RenderLock(groups...)
	// The update mode: TESTKIT_LOCK_WRITE=1 seeds or rewrites the
	// manifest from the emitted set, for the regeneration workflow —
	// the diff is then reviewed in version control, the same review
	// the mismatch message asks for. CI never sets it: a gate that
	// rewrites its own expectation gates nothing.
	if os.Getenv("TESTKIT_LOCK_WRITE") == "1" {
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			tb.Fatalf("write the manifest at %s: %v", path, err)
		}
		tb.Logf("manifest written: %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("no manifest at %s (%v).\n"+
			"  seed it from the emitted set — TESTKIT_LOCK_WRITE=1 writes it, or\n"+
			"  write RenderLock's output there — and review it as the assertion\n"+
			"  list it is.",
			path, err)
	}
	if string(want) != got {
		tb.Errorf("%s no longer describes the checks this package emits.\n"+
			"regenerate it, and read the diff as the assertion diff it is:\n\n%s",
			path, DiffLock(string(want), got))
	}
}

// VerifyIndex holds a typed index to the emitted set: every ID the index
// offers is a check the suite runs, and every check the suite runs is
// reachable through the index. An index method whose check no longer
// exists compiles fine and points at nothing; a check the index forgot
// cannot be dropped, proven, or run alone.
func VerifyIndex(tb testing.TB, name string, indexed, emitted []ID) {
	tb.Helper()
	inIndex := map[ID]bool{}
	for _, id := range indexed {
		if inIndex[id] {
			tb.Errorf("%s: the index lists %q twice", name, id)
		}
		inIndex[id] = true
	}
	inSuite := map[ID]bool{}
	for _, id := range emitted {
		inSuite[id] = true
		if !inIndex[id] {
			tb.Errorf("%s: the suite emits %q and the index does not name it, "+
				"so it cannot be dropped, proven, or run alone", name, id)
		}
	}
	for _, id := range indexed {
		if !inSuite[id] {
			tb.Errorf("%s: the index names %q and the suite does not emit it; "+
				"the index method compiles and points at nothing", name, id)
		}
	}
}

// DropHinter renders drops the way the runner's failure messages teach
// them: the typed index expression when the table knows the ID, the
// quoted string form otherwise — the fallback a consumer's own check
// legitimately takes, since an ID they authored has no index entry.
// One constructor, so the fallback rule [VerifyDropHints] polices has
// one implementation rather than one per generated package.
func DropHinter(veneer string, paths map[ID]string) func(ID) string {
	return func(id ID) string {
		if path, ok := paths[id]; ok {
			return veneer + ".Without(" + path + ")"
		}
		return veneer + ".Without(" + strconv.Quote(string(id)) + ")"
	}
}

// VerifyDropHints holds the drop-hint table to the emitted set: every
// generated ID must render as its typed index expression. The runner's
// failure messages teach drops through these hints, and the string form
// compiles too while forfeiting the compile-break-on-regeneration the
// index exists for — so a missing table entry is a silent downgrade of
// exactly the protection the index was built to give. The fallback form
// quotes the ID and a typed path never does, which is what makes the
// downgrade detectable rather than plausible.
func VerifyDropHints(tb testing.TB, name string, ids []ID, hint func(ID) string) {
	tb.Helper()
	if hint == nil {
		tb.Errorf("%s: the suite has no drop-hint renderer; set DropHint in the generated suite", name)
		return
	}
	for _, id := range ids {
		if strings.Contains(hint(id), strconv.Quote(string(id))) {
			tb.Errorf("%s: %q has no index path; its drop hint teaches the string form, "+
				"which forfeits the compile-break a regeneration owes", name, id)
		}
	}
}

// VerifyDistinctIDs fails when two suites in one package claim the same
// ID. Scope segments are method names, so two interfaces sharing a method
// name would collide in the ID space; this reports the collision where it
// is introduced rather than leaving it to be discovered at random.
func VerifyDistinctIDs(tb testing.TB, groups ...[]ID) {
	tb.Helper()
	seen := map[ID]bool{}
	for _, g := range groups {
		for _, id := range g {
			if seen[id] {
				tb.Errorf("check ID %q is claimed by more than one interface in this package", id)
			}
			seen[id] = true
		}
	}
}
