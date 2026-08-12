// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ref_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/ref"
)

// TestCollection pins what the stream laws lean on: insertion order kept,
// duplicates kept, and a drain that hands back a copy.
func TestCollection(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	c := ref.NewCollection[string]()

	got, err := c.Items(ctx)
	testkit.NoError(t, err, "draining an empty collection succeeds")
	testkit.Len(t, got, 0, "with nothing in it")

	testkit.NoError(t, c.Add(ctx, "a"), "first add")
	testkit.NoError(t, c.Add(ctx, "b"), "second add")
	testkit.NoError(t, c.Add(ctx, "a"), "a duplicate is kept, not collapsed")

	got, _ = c.Items(ctx)
	testkit.Equal(t, got, []string{"a", "b", "a"},
		"insertion order and duplicates survive")

	got[0] = "mutated"
	again, _ := c.Items(ctx)
	testkit.Equal(t, again[0], "a", "a drained copy cannot disturb the oracle")
}

// TestSetCollection pins the dedupe: the second identical add collapses, and
// first-insertion order survives.
func TestSetCollection(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	c := ref.NewSetCollection[string]()
	testkit.NoError(t, c.Add(ctx, "a"), "first add")
	testkit.NoError(t, c.Add(ctx, "b"), "second add")
	testkit.NoError(t, c.Add(ctx, "a"), "the repeat succeeds")
	got, err := c.Items(ctx)
	testkit.NoError(t, err, "and drains")
	testkit.Equal(t, got, []string{"a", "b"}, "collapsed, first-insertion order kept")
}
