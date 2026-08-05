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
