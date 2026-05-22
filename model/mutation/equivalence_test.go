// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutation_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/model/mutation"
)

func TestEquivalenceClasses(t *testing.T) {
	t.Parallel()

	t.Run("no reports returns nil", func(t *testing.T) {
		t.Parallel()
		got := mutation.EquivalenceClasses()
		testkit.True(t, got == nil, "nil for empty")
	})

	t.Run("identical kill profiles cluster", func(t *testing.T) {
		t.Parallel()
		r1 := mutation.Report{Results: []mutation.Result{
			{Operator: "a", Killed: true},
			{Operator: "b", Killed: true},
			{Operator: "c", Killed: false},
		}}
		r2 := mutation.Report{Results: []mutation.Result{
			{Operator: "a", Killed: false},
			{Operator: "b", Killed: false},
			{Operator: "c", Killed: true},
		}}
		got := mutation.EquivalenceClasses(r1, r2)
		// a and b share profile {killed, killed→survived, survived}; c is alone.
		testkit.Equal(t, len(got), 2, "two equivalence classes")
		// First class (alphabetical by profile) should be c (profile "01"), then a,b ("10").
		testkit.Equal(t, got[0], []string{"c"}, "c alone")
		testkit.Equal(t, got[1], []string{"a", "b"}, "a and b equivalent")
	})

	t.Run("missing operator treated as survived", func(t *testing.T) {
		t.Parallel()
		r1 := mutation.Report{Results: []mutation.Result{
			{Operator: "a", Killed: true},
		}}
		r2 := mutation.Report{Results: []mutation.Result{
			{Operator: "a", Killed: true},
			{Operator: "b", Killed: false},
		}}
		got := mutation.EquivalenceClasses(r1, r2)
		// "a" profile = "11", "b" profile = "00" (treated as survived in r1).
		testkit.Equal(t, len(got), 2, "two classes")
		testkit.Equal(t, got[0], []string{"b"}, "b unique")
		testkit.Equal(t, got[1], []string{"a"}, "a alone")
	})

	t.Run("single-report run still returns classes", func(t *testing.T) {
		t.Parallel()
		r := mutation.Report{Results: []mutation.Result{
			{Operator: "a", Killed: true},
			{Operator: "b", Killed: false},
			{Operator: "c", Killed: true},
		}}
		got := mutation.EquivalenceClasses(r)
		testkit.Equal(t, len(got), 2, "killed vs survived")
		// b survived → profile "0"; a, c both killed → profile "1"
		testkit.Equal(t, got[0], []string{"b"}, "survivors")
		testkit.Equal(t, got[1], []string{"a", "c"}, "killed pair")
	})

	t.Run("all-killed cluster is one class", func(t *testing.T) {
		t.Parallel()
		r := mutation.Report{Results: []mutation.Result{
			{Operator: "a", Killed: true},
			{Operator: "b", Killed: true},
		}}
		got := mutation.EquivalenceClasses(r)
		testkit.Equal(t, len(got), 1, "single class")
		testkit.Equal(t, got[0], []string{"a", "b"}, "all together")
	})
}
