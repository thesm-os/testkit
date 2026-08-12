// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import (
	"context"
	"sync"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/gate"
	"go.thesmos.sh/testkit/generator/tiers"
)

// assertionState is the one owed-versus-bound measurement this file's tests
// share: an Annotate run for the stamps, an Emission run for the bindings —
// each a full pipeline over the corpus, too expensive to repeat per test.
type assertionState struct {
	owed  map[string]bool
	bound map[string]bool
	err   error
}

//nolint:gochecknoglobals // memoized measurement, test-only.
var assertionOnce = sync.OnceValue(func() assertionState {
	s := assertionState{owed: map[string]bool{}, bound: map[string]bool{}}
	stamped, err := gate.Annotate(context.Background(), corpusRoot, "./corpus/...")
	if err != nil {
		s.err = err
		return s
	}
	emitted, err := gate.Emission(context.Background(), corpusRoot, "./corpus/...")
	if err != nil {
		s.err = err
		return s
	}
	for _, names := range stamped {
		for _, c := range names {
			for _, law := range tiers.LawsFor(c) {
				// An unsound conduct cannot bind by design; the conduct
				// census owns that quarantine, not this register.
				if gate.LawConduct[law].Sound() {
					s.owed[law] = true
				}
			}
		}
	}
	for _, e := range emitted {
		for _, law := range e.Laws {
			s.bound[law] = true
		}
	}
	return s
})

// TestEveryOwedLawIsBoundOrRegistered is the assertion gate the audit
// commissioned: a classification stamped in the corpus selects laws, and
// each selected, sound law must be bound in at least one fixture — or carried
// in [gate.UnboundLaws] with the chokepoint that holds it. The bounded break
// experiment proved what the gap costs: a fixture's whole claim deleted, the
// corpus green. A red line here names the law that would go the same way.
func TestEveryOwedLawIsBoundOrRegistered(t *testing.T) {
	t.Parallel()

	s := assertionOnce()
	if s.err != nil {
		t.Fatalf("measure the corpus: %v", s.err)
	}
	testkit.True(t, len(s.owed) > 0, "the corpus stamps select laws at all")
	for law := range s.owed {
		_, registered := gate.UnboundLaws[law]
		testkit.True(t, s.bound[law] || registered,
			law+" is selected by the corpus and bound nowhere — bind it, or register the debt with its chokepoint")
	}
}

// TestUnboundRegisterOnlyShrinks holds the register to its contract: an entry
// that starts binding must be deleted, and an entry nothing selects any more
// is a zombie. Either way the register moves in one direction.
func TestUnboundRegisterOnlyShrinks(t *testing.T) {
	t.Parallel()

	s := assertionOnce()
	if s.err != nil {
		t.Fatalf("measure the corpus: %v", s.err)
	}
	for law, reason := range gate.UnboundLaws {
		testkit.False(t, s.bound[law],
			law+" now binds — delete its register entry; the debt is paid")
		testkit.True(t, s.owed[law],
			law+" is owed by nothing the corpus stamps — a zombie entry records no debt")
		testkit.True(t, len(reason) > 30,
			law+"'s reason says what it is waiting on")
	}
}

// TestEmissionSeesTheTwinFloor pins the measurement's other axis: the
// reference kind is read off the queued bindings, so the twin count the
// audit hand-derived stays derivable — and a derived fixture regressing to
// the twin is visible here before it is visible nowhere.
func TestEmissionSeesTheTwinFloor(t *testing.T) {
	t.Parallel()

	emitted, err := gate.Emission(t.Context(), corpusRoot,
		"./corpus/iface/mixin/validates", "./corpus/iface/mixin/bounded")
	if err != nil {
		t.Fatalf("measure two fixtures: %v", err)
	}
	kinds := map[string]bool{}
	for _, e := range emitted {
		kinds[e.Fixture] = e.Twin
	}
	testkit.False(t, kinds["go.thesmos.sh/testkit/conformance/corpus/iface/mixin/validates.Mixed"],
		"validates derives the map oracle")
	testkit.True(t, kinds["go.thesmos.sh/testkit/conformance/corpus/iface/mixin/bounded.Mixed"],
		"bounded rides the twin floor — the audit's break experiment, kept measurable")
}

// TestEmissionSurfacesARunFailure pins the error arm: a pattern matching
// nothing is a run that failed, not an empty measurement quietly read as
// "nothing owed".
func TestEmissionSurfacesARunFailure(t *testing.T) {
	t.Parallel()

	_, err := gate.Emission(t.Context(), corpusRoot, "./corpus/definitely-not-here/...")
	testkit.True(t, err != nil, "a failed run reports, never measures empty")
}
