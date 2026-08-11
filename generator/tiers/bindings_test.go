// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers_test

import (
	"slices"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/tiers"
)

// TestBindingFor pins both answers: a filled row comes back whole, and a law
// the column does not cover answers false rather than an empty spec a
// generator would render as a type with no name.
func TestBindingFor(t *testing.T) {
	t.Parallel()

	b, ok := tiers.BindingFor(lawid.WriteObservable)
	testkit.True(t, ok, "the write-observable row is filled")
	testkit.Equal(t, b.Type, "WriteObservable", "and names the struct")
	testkit.Equal(t, len(b.Args), 2, "with its two arguments after the subject")

	_, ok = tiers.BindingFor("AUTO-NOT-A-LAW")
	testkit.False(t, ok, "an uncovered law answers false")
}

// TestBoundNamesDeclaredLaws holds every key of the column to the vocabulary,
// the same both-registries discipline every other tiers table lives under.
func TestBoundNamesDeclaredLaws(t *testing.T) {
	t.Parallel()

	declared := lawid.All()
	for _, id := range tiers.Bound() {
		testkit.True(t, slices.Contains(declared, id),
			id+" is an identifier the vocabulary declares")
	}
	testkit.True(t, slices.IsSorted(tiers.Bound()), "and the listing is sorted")
}
