// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"testing"

	"go.thesmos.sh/testkit/engine/suite"
)

type fake struct{}

func check(id suite.ID) suite.Check[fake] {
	return suite.Check[fake]{
		ID:    id,
		Class: "signature/smoke",
		Claim: "a claim",
		Run:   func(testing.TB, fake) {},
	}
}

func TestSuiteCombinatorsCopy(t *testing.T) {
	t.Parallel()

	base := suite.Suite[fake]{Name: "base", Checks: []suite.Check[fake]{check("A/one")}}

	with := base.With(check("B/two"))
	if len(base.Checks) != 1 {
		t.Errorf("With mutated the receiver: base has %d checks, want 1", len(base.Checks))
	}
	if len(with.Checks) != 2 {
		t.Errorf("With returned %d checks, want 2", len(with.Checks))
	}

	without := with.Without("A/one")
	if without.Dropped("A/one") != true {
		t.Error("Without did not record the drop")
	}
	if with.Dropped("A/one") {
		t.Error("Without mutated the receiver")
	}
	if len(without.Checks) != 2 {
		t.Errorf("Without removed a check from the set; it must only mark it, "+
			"so the report can name it and Run can tell a drop from a typo (got %d)",
			len(without.Checks))
	}
}

// The witnesses are load-bearing exports with no behavior; referencing
// them here records that the zero coverage is intentional.
var (
	_ = suite.CompatV2
)

func TestWithoutClonesTheCapabilityMaps(t *testing.T) {
	t.Parallel()

	s := suite.Suite[int]{Checks: []suite.Check[int]{{
		ID:    "A/one",
		Class: suite.ClassSmoke,
		Needs: suite.Needs("door", 1),
		Run:   func(testing.TB, int) {},
	}}}
	dropped := s.Without("A/one")
	dropped.Checks[0].Needs["door"] = 2
	if got := s.Checks[0].Needs["door"]; got != 1 {
		t.Errorf("a clone's capability map must not reach back into the original, got %v", got)
	}
}

func TestReferenceOnAnUndeclaredOracle(t *testing.T) {
	t.Parallel()

	var sub suite.Subject[int]
	if _, ok := sub.Reference(); ok {
		t.Error("a run with no oracle must answer ok=false, not invent a constructor")
	}
}
