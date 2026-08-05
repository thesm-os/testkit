// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutation_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/mutation"
)

// fakeOp is a minimal Operator used by harness tests; the harness
// doesn't actually call any of its decision methods.
type fakeOp struct{ n string }

func (f fakeOp) Name() string { return f.n }

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("operator marked killed when runWith reports failure", func(t *testing.T) {
		t.Parallel()
		op := fakeOp{n: "killer"}
		report := mutation.Run(t, []mutation.Operator{op}, func(tb testing.TB, _ mutation.Operator) {
			tb.Helper()
			tb.Errorf("synthetic failure")
		})
		testkit.Equal(t, len(report.Results), 1, "one result")
		testkit.Equal(t, report.Results[0].Killed, true, "killed")
		testkit.Assert(t, report.Results[0].FailureMsg).
			Contains("synthetic failure", "msg preserved")
	})

	t.Run("operator marked survived when runWith does not fail", func(t *testing.T) {
		t.Parallel()
		op := fakeOp{n: "survivor"}
		report := mutation.Run(t, []mutation.Operator{op}, func(_ testing.TB, _ mutation.Operator) {})
		testkit.Equal(t, len(report.Results), 1, "one result")
		testkit.Equal(t, report.Results[0].Killed, false, "survived")
		testkit.Equal(t, report.Results[0].FailureMsg, "", "no failure msg")
	})

	t.Run("Result order matches operators argument order", func(t *testing.T) {
		t.Parallel()
		ops := []mutation.Operator{
			fakeOp{n: "first"},
			fakeOp{n: "second"},
			fakeOp{n: "third"},
		}
		report := mutation.Run(t, ops, func(_ testing.TB, _ mutation.Operator) {})
		testkit.Equal(t, report.Results[0].Operator, "first", "stable order 1")
		testkit.Equal(t, report.Results[1].Operator, "second", "stable order 2")
		testkit.Equal(t, report.Results[2].Operator, "third", "stable order 3")
	})
}

func TestReport(t *testing.T) {
	t.Parallel()

	t.Run("KillRate computes killed / total", func(t *testing.T) {
		t.Parallel()
		report := mutation.Report{
			Results: []mutation.Result{
				{Operator: "a", Killed: true},
				{Operator: "b", Killed: false},
				{Operator: "c", Killed: true},
				{Operator: "d", Killed: false},
			},
		}
		testkit.Equal(t, report.KillRate(), 0.5, "2 of 4")
	})

	t.Run("empty Report has zero kill rate", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, mutation.Report{}.KillRate(), 0.0, "empty")
	})

	t.Run("Unkilled returns survivors in sorted order", func(t *testing.T) {
		t.Parallel()
		report := mutation.Report{
			Results: []mutation.Result{
				{Operator: "zebra", Killed: false},
				{Operator: "alpha", Killed: false},
				{Operator: "beta", Killed: true},
			},
		}
		testkit.Equal(t, report.Unkilled(), []string{"alpha", "zebra"}, "sorted")
	})

	t.Run("Killed returns killed ops in sorted order", func(t *testing.T) {
		t.Parallel()
		report := mutation.Report{
			Results: []mutation.Result{
				{Operator: "zebra", Killed: true},
				{Operator: "alpha", Killed: true},
				{Operator: "beta", Killed: false},
			},
		}
		testkit.Equal(t, report.Killed(), []string{"alpha", "zebra"}, "sorted")
	})

	t.Run("Killed returns empty when none killed", func(t *testing.T) {
		t.Parallel()
		report := mutation.Report{
			Results: []mutation.Result{{Operator: "a", Killed: false}},
		}
		testkit.Equal(t, len(report.Killed()), 0, "empty")
	})

	t.Run("Unkilled returns empty when all killed", func(t *testing.T) {
		t.Parallel()
		report := mutation.Report{
			Results: []mutation.Result{{Operator: "a", Killed: true}},
		}
		testkit.Equal(t, len(report.Unkilled()), 0, "empty")
	})
}
