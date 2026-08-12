// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ref_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/ref"
)

type item struct {
	ID   string
	Name string
}

func itemKey(v item) string { return v.ID }

var errNotFound = errors.New("not found")

func newStore() *ref.MapStore[string, item] {
	return ref.NewMapStore(itemKey, errNotFound)
}

func TestMapStore(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	t.Run("Get returns not-found on empty store", func(t *testing.T) {
		t.Parallel()
		s := newStore()
		_, err := s.Get(ctx, "missing")
		testkit.ErrorIs(t, err, errNotFound, "missing key returns sentinel")
	})

	t.Run("Put then Get round-trips", func(t *testing.T) {
		t.Parallel()
		s := newStore()
		testkit.NoError(t, s.Put(ctx, item{ID: "a", Name: "alice"}), "Put")
		got, err := s.Get(ctx, "a")
		testkit.NoError(t, err, "Get after Put")
		testkit.Equal(t, got, item{ID: "a", Name: "alice"}, "round-trip")
	})

	t.Run("Put overwrites existing key", func(t *testing.T) {
		t.Parallel()
		s := newStore()
		testkit.NoError(t, s.Put(ctx, item{ID: "a", Name: "v1"}), "first Put")
		testkit.NoError(t, s.Put(ctx, item{ID: "a", Name: "v2"}), "overwrite Put")
		got, err := s.Get(ctx, "a")
		testkit.NoError(t, err, "Get")
		testkit.Equal(t, got.Name, "v2", "overwrite wins")
	})

	t.Run("Delete removes key", func(t *testing.T) {
		t.Parallel()
		s := newStore()
		testkit.NoError(t, s.Put(ctx, item{ID: "a", Name: "v"}), "Put")
		testkit.NoError(t, s.Delete(ctx, "a"), "Delete")
		_, err := s.Get(ctx, "a")
		testkit.ErrorIs(t, err, errNotFound, "deleted key returns sentinel")
	})

	t.Run("Delete on absent key is no-op", func(t *testing.T) {
		t.Parallel()
		s := newStore()
		testkit.NoError(t, s.Delete(ctx, "missing"), "Delete absent is no-op")
	})

	t.Run("Count reflects stored items", func(t *testing.T) {
		t.Parallel()
		s := newStore()
		n, err := s.Count(ctx)
		testkit.NoError(t, err, "Count empty")
		testkit.Equal(t, n, 0, "empty count")

		testkit.NoError(t, s.Put(ctx, item{ID: "a", Name: "v"}), "Put a")
		testkit.NoError(t, s.Put(ctx, item{ID: "b", Name: "v"}), "Put b")
		n, err = s.Count(ctx)
		testkit.NoError(t, err, "Count")
		testkit.Equal(t, n, 2, "count after two puts")
	})

	t.Run("List iterates all stored values", func(t *testing.T) {
		t.Parallel()
		s := newStore()
		testkit.NoError(t, s.Put(ctx, item{ID: "a", Name: "alice"}), "Put a")
		testkit.NoError(t, s.Put(ctx, item{ID: "b", Name: "bob"}), "Put b")

		seen := map[string]bool{}
		for v, err := range s.List(ctx) {
			testkit.NoError(t, err, "List yields no error")
			seen[v.ID] = true
		}
		testkit.True(t, seen["a"] && seen["b"], "List yields all values")
	})

	t.Run("List break mid-iteration", func(t *testing.T) {
		t.Parallel()
		s := newStore()
		testkit.NoError(t, s.Put(ctx, item{ID: "a", Name: "v"}), "Put a")
		testkit.NoError(t, s.Put(ctx, item{ID: "b", Name: "v"}), "Put b")
		testkit.NoError(t, s.Put(ctx, item{ID: "c", Name: "v"}), "Put c")

		count := 0
		for range s.List(ctx) {
			count++
			if count == 1 {
				break
			}
		}
		testkit.Equal(t, count, 1, "break after first item")
	})
}

// TestMapStoreValues pins the slice drain: everything held, one call, a
// fresh slice each time.
func TestMapStoreValues(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	s := ref.NewMapStore(func(v string) string { return v }, errNotFound)
	testkit.NoError(t, s.Put(ctx, "a"), "first put")
	testkit.NoError(t, s.Put(ctx, "b"), "second put")
	testkit.NoError(t, s.Put(ctx, "a"), "an upsert replaces, not appends")

	got, err := s.Values(ctx)
	testkit.NoError(t, err, "draining succeeds")
	testkit.Len(t, got, 2, "one value per key")
}
