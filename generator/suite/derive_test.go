// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"testing"

	"go.thesmos.sh/eidos/lang/golang"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/subject"
	"go.thesmos.sh/testkit/generator/suite"
)

func TestRegistry(t *testing.T) {
	t.Parallel()

	t.Run("every deriver is named", func(t *testing.T) {
		t.Parallel()
		for _, d := range suite.Registry() {
			testkit.NotEqual(t, d.Name(), suite.DeriverName(""),
				"an unnamed deriver cannot be attributed in refusals")
		}
	})

	t.Run("no name registers twice", func(t *testing.T) {
		t.Parallel()
		seen := map[suite.DeriverName]bool{}
		for _, d := range suite.Registry() {
			testkit.False(t, seen[d.Name()], "deriver "+string(d.Name())+" must register once")
			seen[d.Name()] = true
		}
	})
}

// The inventory is every deriver's answer, folded once.
//
// The seam that makes the derive layer live: before it, each deriver
// was reachable only from its own test. What it must not do is drop an
// answer — a family silently missing from the inventory is a check the
// index never names and the consumer never knows was owed.
func TestInventoryOfFoldsEveryDeriver(t *testing.T) {
	t.Parallel()

	iface := suite.Iface{
		Name: "Log", Token: "log", Qualifier: "log",
		Methods: []subject.Method{{Sig: &golang.Sig{Name: "Append"}}},
	}

	inv, _ := suite.InventoryOf(iface)

	testkit.Equal(t, inv.Iface, "Log", "the inventory carries the interface it is for")
	testkit.Equal(t, inv.Token, "log", "and the token every identifier is qualified by")
	testkit.True(t, len(inv.Checks) > 0, "a method with a signature derives at least its smoke")

	// Against the registry rather than a count: a deriver added to the
	// registry and not to the fold is exactly the regression this
	// guards, and a pinned number would pass right through it.
	var direct int
	for _, d := range suite.Registry() {
		plans, _ := d.Derive(iface)
		direct += len(plans)
	}
	testkit.Equal(t, len(inv.Checks), direct,
		"the fold is every deriver's plans and nothing dropped between them")
}
