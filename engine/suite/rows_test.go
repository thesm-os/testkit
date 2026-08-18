// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/testkit/engine/suite"
)

func storeishMethods() suite.NameSet {
	return suite.NewNameSet("Store", "Put", "Get", "Len", "Close")
}

func TestNameSet(t *testing.T) {
	t.Parallel()

	s := storeishMethods()
	if s.Owner() != "Store" {
		t.Errorf("Owner = %q, want the name the set was built for", s.Owner())
	}
	if !s.Has("Put") || s.Has("Delete") {
		t.Error("Has must answer for members and refuse strangers")
	}
	if got := s.List(); got != "Close, Get, Len, Put" {
		t.Errorf("List must render sorted for a stable message, got %q", got)
	}
}

// TestRowID pins the two rules a method-scoped row obeys, and that each
// failure names the field the consumer set — the fix belongs to the field.
func TestRowID(t *testing.T) {
	t.Parallel()

	id, err := suite.RowID("Run", "Put", "newer-wins", storeishMethods())
	if err != nil || id != "Put/newer-wins" {
		t.Errorf("a valid row composes its method ID, got (%q, %v)", id, err)
	}

	_, err = suite.RowID("Prop", "", "nameless", storeishMethods())
	if err == nil || !strings.Contains(err.Error(), "Prop") {
		t.Errorf("a missing Method must fail naming the body field, got %v", err)
	}

	_, err = suite.RowID("Run", "Delete", "typo", storeishMethods())
	if err == nil || !strings.Contains(err.Error(), "Store") || !strings.Contains(err.Error(), "Close, Get, Len, Put") {
		t.Errorf("an unknown method must fail naming the owner and its methods, got %v", err)
	}
}

func TestHandRowID(t *testing.T) {
	t.Parallel()

	if got := suite.HandRowID("no-shared-state"); got != "own/no-shared-state" {
		t.Errorf("a scopeless row lives in the hand-written family, got %q", got)
	}
}

func TestOneBody(t *testing.T) {
	t.Parallel()

	if err := suite.OneBody("row", 1, "Run, Prop"); err != nil {
		t.Errorf("exactly one body passes, got %v", err)
	}
	for _, n := range []int{0, 2} {
		err := suite.OneBody("row", n, "Run, Prop")
		if err == nil || !strings.Contains(err.Error(), "Run, Prop") {
			t.Errorf("%d bodies must fail offering the interface's own fields, got %v", n, err)
		}
	}
}

// TestFalsify pins the lowering truth table: the defect is the claim and
// the evidence in one field, holding both a defect and an argument is
// refused, and neither means unproven.
func TestFalsify(t *testing.T) {
	t.Parallel()

	_, err := suite.Falsify("row", true, "also argued")
	if err == nil || !strings.Contains(err.Error(), "drop Argued") {
		t.Errorf("a row with both must be refused with the fix, got %v", err)
	}

	f, err := suite.Falsify("row", true, "")
	if err != nil || f.State != suite.FalsifiableProven {
		t.Errorf("a defect lowers to Proven, got (%+v, %v)", f, err)
	}

	f, err = suite.Falsify("row", false, "cannot be staged")
	if err != nil || f.State != suite.FalsifiableArgued || f.Why != "cannot be staged" {
		t.Errorf("an argument lowers to Argued carrying its why, got (%+v, %v)", f, err)
	}

	f, err = suite.Falsify("row", false, "")
	if err != nil || f.State != "" {
		t.Errorf("neither is the zero value, which encodes unproven, got (%+v, %v)", f, err)
	}
}

func TestProvenCheck(t *testing.T) {
	t.Parallel()

	ran := false
	c := suite.ProvenCheck[int]("Get/smoke", suite.ClassSmoke, "Get survives",
		func(testing.TB, int) { ran = true })
	if c.ID != "Get/smoke" || c.Class != suite.ClassSmoke || c.Claim != "Get survives" {
		t.Errorf("the row must carry what was passed: %+v", c)
	}
	if c.Falsifiable.State != suite.FalsifiableProven {
		t.Errorf("the sugar's whole point is the Proven stamp, got %q", c.Falsifiable.State)
	}
	c.Run(t, 0)
	if !ran {
		t.Error("the body must be the one given")
	}
}
