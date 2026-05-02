// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/testdata/stub"
)

func TestInMemoryStore(t *testing.T) {
	t.Parallel()

	t.Run("Put and Get round-trip", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()
		item := stub.Item{ID: "order-1", Data: []byte("data")}

		err := s.Put(t.Context(), item)
		testkit.NoError(t, err, "Put must succeed")

		got, err := s.Get(t.Context(), "order-1")
		testkit.NoError(t, err, "Get must succeed")
		testkit.Equal(t, got, item, "must return stored item")
	})

	t.Run("Get missing returns ErrNotFound", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()

		_, err := s.Get(t.Context(), "nonexistent")
		testkit.ErrorIs(t, err, stub.ErrNotFound, "must return ErrNotFound")
	})

	t.Run("Put duplicate returns ErrConflict", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()
		item := stub.Item{ID: "order-1"}

		testkit.NoError(t, s.Put(t.Context(), item), "first Put")
		testkit.ErrorIs(t, s.Put(t.Context(), item), stub.ErrConflict, "duplicate Put")
	})

	t.Run("Delete removes item", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()
		item := stub.Item{ID: "order-1"}

		testkit.NoError(t, s.Put(t.Context(), item), "Put")
		testkit.NoError(t, s.Delete(t.Context(), "order-1"), "Delete")

		_, err := s.Get(t.Context(), "order-1")
		testkit.ErrorIs(t, err, stub.ErrNotFound, "deleted must not be found")
	})

	t.Run("Count returns number of items", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()

		testkit.Equal(t, s.Count(t.Context()), 0, "empty store")
		testkit.NoError(t, s.Put(t.Context(), stub.Item{ID: "a"}), "Put a")
		testkit.NoError(t, s.Put(t.Context(), stub.Item{ID: "b"}), "Put b")
		testkit.Equal(t, s.Count(t.Context()), 2, "two items")
	})

	t.Run("Find retrieves multiple items", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()
		testkit.NoError(t, s.Put(t.Context(), stub.Item{ID: "a"}), "Put a")
		testkit.NoError(t, s.Put(t.Context(), stub.Item{ID: "b"}), "Put b")

		items, err := s.Find(t.Context(), "a", "b")
		testkit.NoError(t, err, "Find must succeed")
		testkit.Len(t, items, 2, "must find both")
	})

	t.Run("Find missing returns ErrNotFound", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()

		_, err := s.Find(t.Context(), "nonexistent")
		testkit.ErrorIs(t, err, stub.ErrNotFound, "must return ErrNotFound")
	})

	t.Run("PutMany stores multiple items atomically", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()

		err := s.PutMany(t.Context(), stub.Item{ID: "a"}, stub.Item{ID: "b"})
		testkit.NoError(t, err, "PutMany must succeed")
		testkit.Equal(t, s.Count(t.Context()), 2, "must store both")
	})

	t.Run("GetOptional returns nil for missing", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()

		got := s.GetOptional(t.Context(), "nonexistent")
		testkit.True(t, got == nil, "must return nil for missing")
	})

	t.Run("GetOptional returns pointer for existing", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()
		testkit.NoError(t, s.Put(t.Context(), stub.Item{ID: "a", Name: "Alice"}), "Put")

		got := s.GetOptional(t.Context(), "a")
		testkit.True(t, got != nil, "must return non-nil")
		testkit.Equal(t, got.Name, "Alice", "must return correct item")
	})

	t.Run("Ping returns nil", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()
		testkit.NoError(t, s.Ping(t.Context()), "Ping must succeed")
	})

	t.Run("Close returns nil", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()
		testkit.NoError(t, s.Close(), "Close must succeed")
	})

	t.Run("recordings work through domain logic", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()

		testkit.NoError(t, s.Put(t.Context(), stub.Item{ID: "order-1"}), "Put")
		_, err := s.Get(t.Context(), "order-1")
		testkit.NoError(t, err, "Get")

		s.OnPut.AssertCalledOnce(t, "must record Put")
		s.OnGet.AssertCalledOnce(t, "must record Get")
	})

	t.Run("fault injection overrides domain logic", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()
		s.OnPut.Faults(stub.ErrConflict, 1)

		err := s.Put(t.Context(), stub.Item{ID: "new"})
		testkit.ErrorIs(t, err, stub.ErrConflict, "fault must override")

		_, err = s.Get(t.Context(), "new")
		testkit.ErrorIs(t, err, stub.ErrNotFound, "faulted Put must not store")
	})
}
