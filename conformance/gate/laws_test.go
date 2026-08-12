// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/gate"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/engine/model/law"
	"go.thesmos.sh/testkit/generator/tiers"
)

// TestEveryLawIsAccountedFor holds the identifier vocabulary to the catalogue
// that reports it.
//
// The two live in modules that do not depend on each other, which is why the
// check lives here. Before the vocabulary existed each side spelled its own
// literals and the table had drifted to naming thirty-three of eighty-three
// laws with nothing able to see it.
func TestEveryLawIsAccountedFor(t *testing.T) {
	t.Parallel()

	t.Run("every declared identifier names a law that reports it", func(t *testing.T) {
		t.Parallel()
		for _, id := range lawid.All() {
			typ, mapped := gate.LawTypes[id]
			testkit.True(t, mapped, id+" names a law type")
			if !mapped {
				continue
			}
			reported, ok := gate.ReportedID(id, typ)
			testkit.True(t, ok, id+"'s law implements ID")
			testkit.Equal(t, reported, id,
				typ.Name()+" reports the identifier the vocabulary declares for it")
		}
	})

	t.Run("the table knows nothing the vocabulary does not", func(t *testing.T) {
		t.Parallel()
		// The other direction. An entry for an identifier nobody declares is a
		// law the selection rules can never name, since a rule spells its law
		// as a constant.
		declared := lawid.All()
		for id := range gate.LawTypes {
			testkit.True(t, slices.Contains(declared, id),
				id+" is declared in the vocabulary")
		}
	})
}

// TestEveryLawIsSelectedOrExcused is the census docs/adr/0017 asks for, one
// level down from classifications.
//
// A law nothing selects is unreachable: it ships, it is tested in `engine`, and
// no declaration can ever cause it to run. That is a defect when a rule was
// simply not written, and a boundary when nothing can express the selection —
// and the two are the same silence unless the second is recorded.
func TestEveryLawIsSelectedOrExcused(t *testing.T) {
	t.Parallel()

	selected := map[string]bool{}
	for _, r := range tiers.Rules() {
		selected[r.Law] = true
	}

	for _, id := range lawid.All() {
		if selected[id] {
			continue
		}
		reason := gate.UnreachableLaws[id]
		testkit.True(t, reason != "",
			id+" is selected by a rule, or recorded here with what it is waiting on")
	}
}

// TestNoExcuseOutlivesItsLaw stops the census from accumulating stale entries.
//
// An excuse for a law that a rule now selects reads as a boundary that is no
// longer there, and the next person to widen the catalogue trusts it.
func TestNoExcuseOutlivesItsLaw(t *testing.T) {
	t.Parallel()

	selected := map[string]bool{}
	for _, r := range tiers.Rules() {
		selected[r.Law] = true
	}

	for id := range gate.UnreachableLaws {
		testkit.True(t, slices.Contains(lawid.All(), id), id+" is a declared identifier")
		testkit.False(t, selected[id],
			id+" is excused as unreachable, so no rule may select it")
	}
}

// TestEveryManifestMatchesItsLaw is the check the whole manifest exists for.
//
// A rule's Fields say where each of a law's fields comes from, and a binding
// fills exactly those. A manifest naming a field the struct does not have
// fills nothing; a struct field the manifest omits is left at its zero — and a
// nil closure in a law is a check that runs, passes, and asserts nothing.
//
// Neither is visible any other way. The rules compile against a string, the
// law compiles against nothing, and every other test in this repository passes
// either way.
func TestEveryManifestMatchesItsLaw(t *testing.T) {
	t.Parallel()

	for _, r := range tiers.Rules() {
		typ, mapped := gate.LawTypes[r.Law]
		testkit.True(t, mapped, r.Law+" names a law type")
		if !mapped {
			continue
		}

		want := exportedFields(typ)
		got := make([]string, 0, len(r.Fields))
		for _, f := range r.Fields {
			got = append(got, f.Name)
		}
		slices.Sort(got)

		testkit.Equal(t, got, want,
			r.Law+"'s manifest names exactly the fields "+typ.Name()+" declares")
	}
}

// TestEveryManifestNamesEachFieldOnce guards the comparison above.
//
// Sorted equality would accept a manifest naming one field twice and omitting
// another, since the two lists could still match in length and content order.
func TestEveryManifestNamesEachFieldOnce(t *testing.T) {
	t.Parallel()

	for _, r := range tiers.Rules() {
		seen := map[string]bool{}
		for _, f := range r.Fields {
			testkit.False(t, seen[f.Name], r.Law+" names "+f.Name+" once")
			seen[f.Name] = true
		}
	}
}

// exportedFields returns the struct's exported field names, sorted.
//
// Unexported fields are excluded because a binding cannot reach them: the
// stateful laws keep their memory in one, and a manifest naming it would be
// describing something no generated file can fill.
func exportedFields(t reflect.Type) []string {
	out := make([]string, 0, t.NumField())
	for f := range t.Fields() {
		if f.IsExported() {
			out = append(out, f.Name)
		}
	}
	slices.Sort(out)
	return out
}

// TestUnreachableReasonsAreActionable keeps the census from degenerating into
// a list of shrugs.
//
// The reason is what a reader acts on: it has to name the thing that would
// have to exist first, not merely assert that something does not.
func TestUnreachableReasonsAreActionable(t *testing.T) {
	t.Parallel()

	for id, reason := range gate.UnreachableLaws {
		testkit.True(t, len(reason) > 30, id+"'s reason says what it is waiting on")
		testkit.False(t, strings.HasSuffix(reason, "."),
			id+"'s reason reads as a clause in the report that prints it")
	}
}

// TestEveryBindingRowMatchesItsLaw holds the instantiation column to the
// shipped structs, both halves.
//
// The type name is transcription and a transcription can drift —
// AUTO-CURSOR-NEXT-AFTER-CLOSE's struct is CursorNextAfterCloseSentinel, so
// nothing mechanical checks a row but this. The argument count is the sharper
// edge: WriteObservable is [T, V, K] where ReadAfterWrite is [T, K, V], and a
// row with the wrong arity renders a file that fails to compile in whichever
// corpus package arms it first — after the generator ran clean.
func TestEveryBindingRowMatchesItsLaw(t *testing.T) {
	t.Parallel()

	for _, id := range tiers.Bound() {
		b, _ := tiers.BindingFor(id)
		typ, known := gate.LawTypes[id]
		testkit.True(t, known, id+" is a law the census maps")
		if !known {
			continue
		}

		name, args, instantiated := strings.Cut(typ.Name(), "[")
		testkit.Equal(t, b.Type, name,
			id+"'s row names the struct that reports it")
		testkit.True(t, instantiated, id+"'s census entry is instantiated")
		testkit.Equal(t, len(b.Args)+1, strings.Count(args, ",")+1,
			id+"'s row supplies one argument per type parameter after the subject")

		// A stateful law keeps memory behind a pointer receiver, so its value
		// type has no Check — and a row that gets Ptr wrong renders a literal
		// that fails to compile in whichever package arms it first.
		_, valueHasCheck := typ.MethodByName("Check")
		testkit.Equal(t, b.Ptr, !valueHasCheck,
			id+"'s row addresses the literal exactly when the law is stateful")
	}
}

// TestEveryStatefulLawResets holds the memory-carrying laws to
// [law.Resettable]: cross-action state lives in unexported fields, the pair
// is rebuilt fresh every property iteration, and a law the runner cannot
// reset false-fails the first iteration whose draws differ from the last —
// the leak the wide pools surfaced. Trace-bound laws are exempt: their one
// unexported field is rebound per iteration through [law.TraceBinder].
func TestEveryStatefulLawResets(t *testing.T) {
	t.Parallel()

	resettable := reflect.TypeFor[law.Resettable]()
	binder := reflect.TypeFor[law.TraceBinder]()
	for id, typ := range gate.LawTypes {
		stateful := false
		for f := range typ.Fields() {
			if !f.IsExported() {
				stateful = true
			}
		}
		if !stateful || reflect.PointerTo(typ).Implements(binder) {
			continue
		}
		testkit.True(t, reflect.PointerTo(typ).Implements(resettable),
			id+" carries cross-action state the runner must reset per iteration")
	}
}

// TestReportedIDDeclinesANonLaw covers the arm the table itself cannot reach.
//
// Every entry in [gate.LawTypes] implements ID, so the refusal is unreachable
// from the census — and it is the arm that stops a type added without the
// method from being read as an empty identifier that matches nothing.
func TestReportedIDDeclinesANonLaw(t *testing.T) {
	t.Parallel()

	type notALaw struct{ Field string }

	got, ok := gate.ReportedID("AUTO-NOT-A-LAW", reflect.TypeFor[notALaw]())
	testkit.False(t, ok, "a type with no ID method is declined")
	testkit.Equal(t, got, "", "and reports no identifier")
}
