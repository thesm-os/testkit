// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/testkit/conformance/gate"
)

// This is the second half of the gate, and it exists because the first half
// cannot see what it measures.
//
// A directive naming a sibling — a contract's partner role, a mixin's `fn` —
// records the identifier the author wrote. eidos's contract resolver runs a
// priority bucket later and rewrites it into a qualified name, which is the
// only form a generator can turn back into a call. Register the shape
// classifier without its resolver and every one of those references stays raw:
// coverage still reports complete, because the callable declaring the contract
// stamped it either way, and nothing else reports at all.
//
// That configuration shipped. Twenty-two references in this corpus were raw and
// no test in the repository could tell.
func TestCorpusResolvesEverySiblingReference(t *testing.T) {
	t.Parallel()

	unresolved, err := gate.Resolution(t.Context(), corpusRoot, "./corpus/...")
	if err != nil {
		t.Fatalf("resolve the corpus: %v", err)
	}
	if len(unresolved) == 0 {
		return
	}

	lines := make([]string, len(unresolved))
	for i, u := range unresolved {
		lines[i] = "  " + u.String()
	}
	t.Fatalf("%d sibling reference(s) were not resolved:\n%s\n\n"+
		"The shape resolver is registered by generator.Annotators alongside the "+
		"classifier. A reference can also stay raw because the sibling it names "+
		"is absent or misspelled, which the run reports as a diagnostic.",
		len(unresolved), strings.Join(lines, "\n"))
}

// The corpus is the case where resolution succeeds, so a run over it exercises
// none of the reporting. Asserting the traversal reached something separates
// "everything resolved" from "nothing was read", which look identical from the
// result alone.
func TestResolutionReadsTheCorpus(t *testing.T) {
	t.Parallel()

	// A pattern scoped to one fixture is enough: what is being distinguished is
	// a traversal that ran from one that found no callables at all, and the
	// error path below covers the latter.
	if _, err := gate.Resolution(t.Context(), corpusRoot, "./corpus/iface/contract/lease"); err != nil {
		t.Fatalf("a root-relative pattern must resolve: %v", err)
	}
}

// A pattern matching nothing is a caller mistake, and reporting it as "no
// unresolved references" would read as a pass.
func TestResolutionReportsUnresolvablePattern(t *testing.T) {
	t.Parallel()

	_, err := gate.Resolution(t.Context(), corpusRoot, "./corpus/definitely-not-here/...")
	if err == nil {
		t.Fatal("a pattern matching nothing must be reported")
	}
	if !strings.Contains(err.Error(), "gate:") {
		t.Errorf("the error must name its origin, got: %v", err)
	}
}
