// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package lawid_test

import (
	"slices"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
)

// TestEveryIDIsWellFormed holds the values to the shape the rest of the system
// reads them by.
//
// The prefix is not decoration: `AUTO-` is what distinguishes a law derived
// from a declaration from one a consumer wrote, and the model runner's coverage
// report groups on it. A duplicate value is the quieter defect — two laws with
// one identity, where any map keyed by ID keeps whichever was added last and
// reports the other as never having run.
func TestEveryIDIsWellFormed(t *testing.T) {
	t.Parallel()

	t.Run("every identifier is an AUTO- constant in upper kebab case", func(t *testing.T) {
		t.Parallel()
		for _, id := range lawid.All() {
			testkit.Assert(t, id).HasPrefix("AUTO-", id+" is marked as derived")
			testkit.Equal(t, id, strings.ToUpper(id), id+" is upper case")
			testkit.False(t, strings.Contains(id, "_"), id+" separates on hyphens")
		}
	})

	t.Run("no two identifiers share a value", func(t *testing.T) {
		t.Parallel()
		seen := map[string]bool{}
		for _, id := range lawid.All() {
			testkit.False(t, seen[id], id+" is declared once")
			seen[id] = true
		}
	})

	t.Run("All is sorted", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, slices.IsSorted(lawid.All()),
			"All is sorted, so a reader can diff it against a sorted census")
	})
}
