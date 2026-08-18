// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/testkit/engine/suite"
)

func TestValidateID(t *testing.T) {
	t.Parallel()

	ok := []suite.ID{
		"Put/smoke",
		"Get/zero-on-error",
		"model/AUTO-TTL-EXPIRY",
		"chain/AUTO-STREAM-CHAIN-LINKS",
		"own/hand-written-check",
		"Put/nested/deeper",
		"Get/miss-2",
	}
	for _, id := range ok {
		if err := suite.ValidateID(id); err != nil {
			t.Errorf("ValidateID(%q) = %v, want nil", id, err)
		}
	}

	bad := map[suite.ID]string{
		"":          "empty",
		"Put":       "only a scope segment",
		"model":     "only a scope segment",
		"unknown/x": "neither an exported method name nor a known family",
		"Put/":      "empty segment",

		// The sub-segment grammar. A sentence is the shape the first draft
		// generated and the shape a lock file cannot afford: prose is edited,
		// and the ID is what a Without() call and a checks.lock row are
		// written against. Claim carries the sentence instead.
		"Put/reports a cancelled context": "is not a slug",
		"Put/a\tb":                        "is not a slug",
		"Put/Cancel":                      "is not a slug",

		// A law segment keeps its upper case, because that is the engine's
		// own name for it and the report prints it verbatim. Anything else
		// upper-case is a slug that was not slugged.
		"model/AUTO-TTL_EXPIRY": "may hold only A-Z, 0-9 and '-'",
	}
	for id, want := range bad {
		err := suite.ValidateID(id)
		if err == nil {
			t.Errorf("ValidateID(%q) = nil, want an error about %q", id, want)
			continue
		}
		if !strings.Contains(err.Error(), want) {
			t.Errorf("ValidateID(%q) = %q, want it to mention %q", id, err, want)
		}
	}
}

// TestMethodScopeNeverCollidesWithFamily is the property the case split
// exists for: Go only exports capitalised names, so a method can never
// occupy a family's scope.
func TestMethodScopeNeverCollidesWithFamily(t *testing.T) {
	t.Parallel()

	for _, family := range suite.FamilyNames() {
		// A method named like a family, as Go would export it. A family
		// containing '-' cannot be a Go identifier at all, so for those
		// the collision is impossible for a stronger reason than case.
		exported := strings.ToUpper(family[:1]) + family[1:]
		if !strings.Contains(family, "-") {
			id := suite.ID(exported + "/smoke")
			if err := suite.ValidateID(id); err != nil {
				t.Errorf("a method named %q must be a valid scope: %v", exported, err)
			}
		}
		if suite.IsFamily(exported) {
			t.Errorf("family %q collides with the exported form of itself", family)
		}
	}
}

// TestIDConstructors pins the composition the generated index uses: the
// values must be exactly the strings the lock file records, and the
// qualifier is what A18 requires of a package with more than one
// model-bearing interface.
func TestIDConstructors(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		got  suite.ID
		want suite.ID
	}{
		{suite.MethodID("Put", suite.SegSmoke), "Put/smoke"},
		{suite.MethodID("Get", suite.SegZeroValue), "Get/zero-on-error"},
		{suite.FamilyID(suite.FamilyModel, "store", suite.SegLaws), "model/store/laws"},
		{suite.FamilyID(suite.FamilySim, "store", suite.SegRecovery), "sim/store/recovery"},
		{suite.FamilyID(suite.FamilyModel, "", "AUTO-TTL-EXPIRY"), "model/AUTO-TTL-EXPIRY"},
		{suite.FamilyID(suite.FamilyHand, "", "my-claim"), "own/my-claim"},
	} {
		if tc.got != tc.want {
			t.Errorf("composed %q, want %q", tc.got, tc.want)
		}
		if err := suite.ValidateID(tc.got); err != nil {
			t.Errorf("the composer must produce a valid ID: %v", err)
		}
	}
}

// TestClassesComposeFromFamilies pins that a class and the ID segment
// naming the same leg cannot drift: both read the one Seg constant.
func TestClassesComposeFromFamilies(t *testing.T) {
	t.Parallel()

	if suite.ClassSmoke != "signature/smoke" || suite.ClassZeroValue != "signature/zero-on-error" {
		t.Errorf("the signature classes must keep their lock-file values, got %q %q",
			suite.ClassSmoke, suite.ClassZeroValue)
	}
	if want := suite.Class(suite.FamilyModel + "/" + suite.SegDifferential); suite.ClassDifferential != want {
		t.Errorf("ClassDifferential = %q, want the composed %q", suite.ClassDifferential, want)
	}
	// The class family and the ID family are the same word by construction.
	if suite.ClassFamilyModel != suite.FamilyModel || suite.ClassFamilySim != suite.FamilySim {
		t.Error("a family word must have one home across both vocabularies")
	}
}

// FuzzValidateID drives the grammar with hostile input: IDs land in lock
// files and -run patterns, so the parser must never panic and must judge
// deterministically — and everything the public composers produce from
// an accepted ID's own parts must be accepted back. That round trip is
// what keeps the composers and the validator one grammar rather than
// two.
func FuzzValidateID(f *testing.F) {
	for _, seed := range []string{
		"", "Put/smoke", "Put/zero-on-error", "model/differential",
		"model/store/laws", "model/AUTO-TTL-EXPIRY", "own/hand-written",
		"Put", "put/smoke", "Put/", "/smoke", "Put//x", "model/AUTO-",
		"Put/Smoke", "Put me down/x", "model/\x00", "AUTO-X/y",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		id := suite.ID(s)
		err := suite.ValidateID(id)
		if again := suite.ValidateID(id); (err == nil) != (again == nil) {
			t.Fatalf("ValidateID is not deterministic for %q: %v vs %v", s, err, again)
		}
		if err != nil {
			return
		}
		scope, rest, _ := strings.Cut(s, "/")
		var recomposed suite.ID
		if suite.IsFamily(scope) {
			recomposed = suite.FamilyID(scope, "", rest)
		} else {
			recomposed = suite.MethodID(scope, rest)
		}
		if string(recomposed) != s {
			t.Fatalf("recomposing %q through the public composers changed it to %q", s, recomposed)
		}
		if err := suite.ValidateID(recomposed); err != nil {
			t.Fatalf("the composers produced an ID the validator refuses: %v", err)
		}
	})
}
