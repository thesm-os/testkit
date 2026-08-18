// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"context"
	"slices"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model"
	"go.thesmos.sh/testkit/engine/model/action"
	"go.thesmos.sh/testkit/engine/model/law"
)

// alwaysVacuous declines every check, which is what a law behind a
// precondition the run never supplies does.
type alwaysVacuous struct{ id string }

func (l alwaysVacuous) ID() string  { return l.id }
func (alwaysVacuous) REQID() string { return "" }
func (alwaysVacuous) Check(*rapid.T, storeIface, storeIface) error {
	return law.Vacuous
}

// TestLawCensusEngaged pins the one reading of these counters that is sound.
//
// The absolute counts are not comparable between runs — rapid decides how many
// times a property runs and re-runs it while shrinking — so the only question
// worth asking is whether a law ever reached a verdict.
func TestLawCensusEngaged(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		c    model.LawCensus
		want bool
	}{
		{"never ran", model.LawCensus{}, false},
		{"ran and never declined", model.LawCensus{Ran: 5}, true},
		{"declined every time", model.LawCensus{Ran: 5, Vacuous: 5}, false},
		{"declined all but once", model.LawCensus{Ran: 5, Vacuous: 4}, true},
		{"fired", model.LawCensus{Ran: 5, Fired: 1}, true},
	} {
		if got := tc.c.Engaged(); got != tc.want {
			t.Errorf("%s: Engaged() = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestOutcomeEngaged holds the whole-run verdict to "any law reached a
// verdict", and pins the no-laws case as unengaged rather than as a pass.
func TestOutcomeEngaged(t *testing.T) {
	t.Parallel()

	if (model.Outcome{}).Engaged() {
		t.Error("a run with no laws engaged nothing")
	}
	one := model.Outcome{Laws: map[string]model.LawCensus{
		"A": {Ran: 3, Vacuous: 3},
		"B": {Ran: 3, Vacuous: 2},
	}}
	if !one.Engaged() {
		t.Error("one engaged law engages the run")
	}
	none := model.Outcome{Laws: map[string]model.LawCensus{
		"A": {Ran: 3, Vacuous: 3},
	}}
	if none.Engaged() {
		t.Error("every law declined, so the run proved nothing")
	}
}

// TestOutcomeUnengaged names the laws a reader has to go and fix, sorted so
// the message is stable across runs.
func TestOutcomeUnengaged(t *testing.T) {
	t.Parallel()

	o := model.Outcome{Laws: map[string]model.LawCensus{
		"LAW-Z": {Ran: 3, Vacuous: 3},
		"LAW-A": {Ran: 3, Vacuous: 3},
		"LAW-M": {Ran: 3, Vacuous: 1},
		"LAW-N": {},
	}}
	got := o.Unengaged()
	want := []string{"LAW-A", "LAW-Z"}
	if !slices.Equal(got, want) {
		t.Fatalf("Unengaged() = %v, want %v — engaged and never-ran laws are not findings", got, want)
	}
	if len((model.Outcome{}).Unengaged()) != 0 {
		t.Error("no laws is no findings")
	}
}

// TestRunReturnsTheCensus is the point of the whole type: the datum existed
// inside every run and was discarded at the boundary.
func TestRunReturnsTheCensus(t *testing.T) {
	t.Parallel()

	t.Run("a law that engages is reported engaged", func(t *testing.T) {
		t.Parallel()
		laws := model.NewRegistry[storeIface]()
		laws.Add(law.CountEqualsReference[storeIface, int]{
			Count: func(rt *rapid.T, s storeIface) (int, error) { return s.Count(rt.Context()) },
		})
		var out model.Outcome
		rapid.Check(t, func(rt *rapid.T) {
			out = model.Run(rt, model.Config[storeIface]{
				SUTFactory: func() storeIface { return newStore() },
				RefFactory: func() storeIface { return newStore() },
				Actions:    countActions(),
				Laws:       laws,
			})
		})
		if !out.Engaged() {
			t.Fatalf("the law ran against a correct subject and engaged; census: %+v", out.Laws)
		}
	})

	t.Run("a law declined every time is reported unengaged, and named", func(t *testing.T) {
		t.Parallel()
		laws := model.NewRegistry[storeIface]()
		laws.Add(alwaysVacuous{id: "AUTO-NEVER-ENGAGES"})
		var out model.Outcome
		rapid.Check(t, func(rt *rapid.T) {
			out = model.Run(rt, model.Config[storeIface]{
				SUTFactory: func() storeIface { return newStore() },
				RefFactory: func() storeIface { return newStore() },
				Actions:    countActions(),
				Laws:       laws,
			})
		})
		if out.Engaged() {
			t.Fatal("a law vacuous on every check has engaged nothing")
		}
		if got := out.Unengaged(); !slices.Equal(got, []string{"AUTO-NEVER-ENGAGES"}) {
			t.Fatalf("the run names the law that asserted nothing, got %v", got)
		}
	})

	t.Run("a run with no laws carries no census", func(t *testing.T) {
		t.Parallel()
		var out model.Outcome
		rapid.Check(t, func(rt *rapid.T) {
			out = model.Run(rt, model.Config[storeIface]{
				SUTFactory: func() storeIface { return newStore() },
				RefFactory: func() storeIface { return newStore() },
				Actions:    countActions(),
			})
		})
		if len(out.Laws) != 0 {
			t.Fatalf("no laws registered, so nothing to census: %+v", out.Laws)
		}
		if out.Engaged() {
			t.Error("a pure differential run engages no laws — its oracle is the reference")
		}
	})
}

// TestAssertReturnsTheCensus pins the second entry point to the same
// behaviour: the two are documented as observably identical.
func TestAssertReturnsTheCensus(t *testing.T) {
	t.Parallel()

	laws := model.NewRegistry[storeIface]()
	laws.Add(alwaysVacuous{id: "AUTO-NEVER-ENGAGES"})
	var out model.Outcome
	rapid.Check(t, func(rt *rapid.T) {
		out = model.Assert(rt, func() storeIface { return newStore() },
			model.WithReference[storeIface](func() storeIface { return newStore() }),
			model.WithActions(countActions()...),
			model.WithLaws(laws),
		)
	})
	if out.Engaged() {
		t.Fatal("Assert must report the same unengaged verdict Run does")
	}
}

// countActions is the smallest action set that drives the store: one write and
// one read. Built from the action constructors rather than hand-written
// literals, so the test drives the same vocabulary a generated suite does.
func countActions() []model.Action[storeIface] {
	items := rapid.SampledFrom([]item{{ID: "a", Name: "a"}, {ID: "b", Name: "b"}})
	return []model.Action[storeIface]{
		action.Writer("put", items, func(ctx context.Context, s storeIface, v item) error {
			return s.Put(ctx, v)
		}),
		action.Aggregator("count", func(ctx context.Context, s storeIface) (int, error) {
			return s.Count(ctx)
		}),
	}
}
