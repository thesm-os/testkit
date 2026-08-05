// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/testkit/conformance/gate"
)

// corpusRoot is the module root relative to this package.
const corpusRoot = ".."

// This is the gate. Every classification eidos registers must be stamped by
// something in the corpus, and "stamped" means the annotator produced it — not
// that a directory is named after it.
//
// The distinction is the whole design. A fixture whose directive reads
// `idempotant` sits in a correctly-named folder and would pass any name check;
// here it stamps nothing and reports as a gap.
func TestCorpusCoversEveryClassification(t *testing.T) {
	t.Parallel()

	stamped, err := gate.Annotate(t.Context(), corpusRoot, "./corpus/...")
	if err != nil {
		t.Fatalf("annotate the corpus: %v", err)
	}

	cov := gate.Compare(stamped)
	if !cov.Complete() {
		t.Fatalf("the corpus does not exercise every registered classification:\n%s", cov)
	}
}

// A run that stamps nothing would make the gate above pass only if the
// registries were also empty — but it would also be the symptom of a pipeline
// that silently failed to load the corpus. Asserting a non-trivial result
// separates "nothing is registered" from "nothing was read".
func TestAnnotateReadsTheCorpus(t *testing.T) {
	t.Parallel()

	stamped, err := gate.Annotate(t.Context(), corpusRoot, "./corpus/...")
	if err != nil {
		t.Fatalf("annotate the corpus: %v", err)
	}

	for _, axis := range []string{gate.AxisDetector, gate.AxisContract, gate.AxisMixin} {
		if len(stamped[axis]) == 0 {
			t.Errorf("axis %q stamped nothing; the corpus was not read", axis)
		}
	}
}

// Patterns are relative to root so a caller need not know which directory the
// test process runs from. Passing them through unchanged would make the result
// depend on `go test`'s working directory, which differs between running one
// package and running ./...
func TestAnnotateScopesPatternsToRoot(t *testing.T) {
	t.Parallel()

	if _, err := gate.Annotate(t.Context(), corpusRoot, "./corpus/iface/detector/reader"); err != nil {
		t.Fatalf("a root-relative pattern must resolve: %v", err)
	}
}

// A pattern matching nothing is a caller mistake — a renamed directory, a typo
// — and has to surface. Reporting it as an empty result would let the coverage
// gate above fail with a misleading message about missing fixtures.
func TestAnnotateReportsUnresolvablePattern(t *testing.T) {
	t.Parallel()

	_, err := gate.Annotate(t.Context(), corpusRoot, "./corpus/definitely-not-here/...")
	if err == nil {
		t.Fatal("a pattern matching nothing must be reported")
	}
	if !strings.Contains(err.Error(), "gate:") {
		t.Errorf("the error must name its origin, got: %v", err)
	}
}
