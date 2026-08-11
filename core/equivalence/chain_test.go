// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package equivalence_test

import (
	"reflect"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/equivalence"
)

type entry struct {
	ID        string
	Value     int
	WrittenAt time.Time
	Attempts  int
}

func TestChainEmptyEqualsDeepEquality(t *testing.T) {
	t.Parallel()

	t.Run("identical structs are equal", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain()
		a := entry{ID: "k", Value: 1}
		b := entry{ID: "k", Value: 1}
		testkit.True(t, c.Equal(a, b), "deep equality")
		testkit.Equal(t, c.Diff(a, b), "", "no diff")
	})

	t.Run("differing values diverge", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain()
		a := entry{ID: "k", Value: 1}
		b := entry{ID: "k", Value: 2}
		testkit.False(t, c.Equal(a, b), "diverge")
		testkit.Assert(t, c.Diff(a, b)).Contains("Value", "diff cites field")
	})
}

// TestNilChainIsStrict holds the nil receiver to being the unrefined
// comparison.
//
// Load-bearing rather than defensive. Laws hold a Chain as an optional
// refinement, and a generated binding leaves it unset unless a consumer
// supplied one — so the zero value has to be the correct default. If it
// panicked instead, every such law would need a nil check at each call site,
// and one omission would turn a comparison into a crash mid-run.
func TestNilChainIsStrict(t *testing.T) {
	t.Parallel()

	var c *equivalence.Chain
	a := entry{ID: "k", Value: 1}

	t.Run("equal values compare equal", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, c.Equal(a, entry{ID: "k", Value: 1}), "deep equality")
		testkit.Equal(t, c.Diff(a, entry{ID: "k", Value: 1}), "", "no diff")
	})

	t.Run("differing values diverge and the diff names the field", func(t *testing.T) {
		t.Parallel()
		b := entry{ID: "k", Value: 2}
		testkit.False(t, c.Equal(a, b), "diverge")
		testkit.Assert(t, c.Diff(a, b)).Contains("Value", "diff cites field")
	})

	t.Run("it agrees with an empty chain", func(t *testing.T) {
		t.Parallel()
		empty, b := equivalence.NewChain(), entry{ID: "k", Value: 2}
		testkit.Equal(t, c.Equal(a, b), empty.Equal(a, b),
			"nil and empty are one comparison, not two")
	})
}

func TestChainAddReturnsReceiver(t *testing.T) {
	t.Parallel()

	c := equivalence.NewChain().
		Add(equivalence.Strict()).
		Add(equivalence.IDField(reflect.TypeFor[entry](), "ID"))

	testkit.Equal(t, len(c.Relations()), 2, "two relations registered")
}

func TestChainRelationsCopy(t *testing.T) {
	t.Parallel()

	c := equivalence.NewChain().Add(equivalence.Strict())
	rels := c.Relations()
	testkit.Equal(t, len(rels), 1, "one relation")

	rels[0] = equivalence.Custom("evil", func(_, _ any) bool { return false })
	rels2 := c.Relations()
	testkit.Equal(t, rels2[0].Name(), "strict",
		"mutating returned slice doesn't alter the chain")
}
