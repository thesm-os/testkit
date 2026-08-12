// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ref_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/ref"
)

type chainEntry struct {
	ID   string
	Data string
}

func TestAppendOnlyBasic(t *testing.T) {
	t.Parallel()
	c := ref.NewAppendOnly[chainEntry](nil)
	ctx := t.Context()

	_ = c.Append(ctx, chainEntry{ID: "1", Data: "a"})
	_ = c.Append(ctx, chainEntry{ID: "2", Data: "b"})

	t.Run("replay returns entries in order", func(t *testing.T) {
		t.Parallel()
		var entries []chainEntry
		for e, err := range c.Replay(ctx) {
			if err != nil {
				t.Fatal(err)
			}
			entries = append(entries, e)
		}
		if len(entries) != 2 || entries[0].ID != "1" || entries[1].ID != "2" {
			t.Fatalf("expected [1,2], got %v", entries)
		}
	})

	t.Run("verify passes", func(t *testing.T) {
		t.Parallel()
		err := c.Verify(ctx)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("len", func(t *testing.T) {
		t.Parallel()
		if c.Len() != 2 {
			t.Fatalf("expected 2, got %d", c.Len())
		}
	})

	t.Run("err is nil", func(t *testing.T) {
		t.Parallel()
		if c.Err() != nil {
			t.Fatal("should be nil")
		}
	})
}

func TestPartitionedAppendOnly(t *testing.T) {
	t.Parallel()
	keyOf := func(e chainEntry) string { return e.ID }
	p := ref.NewPartitionedAppendOnly(keyOf, nil)
	ctx := t.Context()

	// Sequential setup — subtests share mutable state.
	_ = p.Append(ctx, chainEntry{ID: "a", Data: "1"})
	_ = p.Append(ctx, chainEntry{ID: "b", Data: "2"})
	_ = p.Append(ctx, chainEntry{ID: "a", Data: "3"})

	t.Run("partition isolation", func(t *testing.T) {
		t.Parallel()
		var aEntries []chainEntry
		for e, err := range p.Replay(ctx, "a") {
			if err != nil {
				t.Fatal(err)
			}
			aEntries = append(aEntries, e)
		}
		if len(aEntries) != 2 {
			t.Fatalf("partition a: expected 2, got %d", len(aEntries))
		}

		var bEntries []chainEntry
		for e, err := range p.Replay(ctx, "b") {
			if err != nil {
				t.Fatal(err)
			}
			bEntries = append(bEntries, e)
		}
		if len(bEntries) != 1 {
			t.Fatalf("partition b: expected 1, got %d", len(bEntries))
		}
	})

	t.Run("verify across partitions", func(t *testing.T) {
		t.Parallel()
		err := p.Verify(ctx)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("partitions are deterministic", func(t *testing.T) {
		t.Parallel()
		parts := p.Partitions()
		if len(parts) != 2 || parts[0] != "a" || parts[1] != "b" {
			t.Fatalf("expected [a,b], got %v", parts)
		}
	})

	t.Run("replay empty partition", func(t *testing.T) {
		t.Parallel()
		var entries []chainEntry
		for e, err := range p.Replay(ctx, "nonexistent") {
			if err != nil {
				t.Fatal(err)
			}
			entries = append(entries, e)
		}
		if len(entries) != 0 {
			t.Fatalf("expected empty, got %v", entries)
		}
	})

	// Err mirrors the single-partition chain: a reference is correct by
	// construction and never latches a poisoned state.
	t.Run("Err reports healthy", func(t *testing.T) {
		t.Parallel()
		if err := p.Err(); err != nil {
			t.Fatalf("a reference chain never poisons, got: %v", err)
		}
	})
}

// The oracle stands opposite subjects held to context respect, so its own
// context-taking operations refuse a cancelled caller the same way.
func TestAppendOnlyRespectsContext(t *testing.T) {
	t.Parallel()

	c := ref.NewAppendOnly[chainEntry](nil)
	testkit.NoError(t, c.Append(t.Context(), chainEntry{ID: "a"}), "a live context appends")

	cancelled, cancel := context.WithCancel(t.Context())
	cancel()

	t.Run("a cancelled append is refused", func(t *testing.T) {
		t.Parallel()
		testkit.ErrorIs(t, c.Append(cancelled, chainEntry{ID: "b"}), context.Canceled,
			"the write never lands")
	})

	t.Run("a cancelled replay yields the context's error", func(t *testing.T) {
		t.Parallel()
		for _, err := range c.Replay(cancelled) {
			testkit.ErrorIs(t, err, context.Canceled, "the drain reports, never streams")
			break
		}
	})

	t.Run("a cancelled verify reports the context's error", func(t *testing.T) {
		t.Parallel()
		testkit.ErrorIs(t, c.Verify(cancelled), context.Canceled,
			"the recomputation never runs")
	})
}
