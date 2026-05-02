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

	t.Run("Get missing item returns ErrNotFound", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()

		_, err := s.Get(t.Context(), "nonexistent")
		testkit.ErrorIs(t, err, stub.ErrNotFound, "must return ErrNotFound")
	})

	t.Run("Put duplicate returns ErrConflict", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()
		item := stub.Item{ID: "order-1"}

		err := s.Put(t.Context(), item)
		testkit.NoError(t, err, "first Put must succeed")

		err = s.Put(t.Context(), item)
		testkit.ErrorIs(t, err, stub.ErrConflict, "duplicate Put must return ErrConflict")
	})

	t.Run("Delete removes item", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()
		item := stub.Item{ID: "order-1"}

		err := s.Put(t.Context(), item)
		testkit.NoError(t, err, "Put must succeed")

		err = s.Delete(t.Context(), "order-1")
		testkit.NoError(t, err, "Delete must succeed")

		_, err = s.Get(t.Context(), "order-1")
		testkit.ErrorIs(t, err, stub.ErrNotFound, "deleted item must not be found")
	})

	t.Run("Delete missing item returns ErrNotFound", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()

		err := s.Delete(t.Context(), "nonexistent")
		testkit.ErrorIs(t, err, stub.ErrNotFound, "must return ErrNotFound")
	})

	t.Run("List returns all items", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()

		err := s.Put(t.Context(), stub.Item{ID: "a"})
		testkit.NoError(t, err, "Put a")
		err = s.Put(t.Context(), stub.Item{ID: "b"})
		testkit.NoError(t, err, "Put b")

		items := s.List(t.Context())
		testkit.Len(t, items, 2, "must return 2 items")
	})

	t.Run("List on empty store returns empty slice", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()
		items := s.List(t.Context())
		testkit.Len(t, items, 0, "empty store must return empty slice")
	})

	t.Run("recordings work through in-memory store", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()

		err := s.Put(t.Context(), stub.Item{ID: "order-1"})
		testkit.NoError(t, err, "Put must succeed")

		_, err = s.Get(t.Context(), "order-1")
		testkit.NoError(t, err, "Get must succeed")

		s.OnPut.AssertCalledOnce(t, "must record Put")
		s.OnGet.AssertCalledOnce(t, "must record Get")

		putCall := s.OnPut.LastCall(t)
		testkit.Equal(t, putCall.Item.ID, "order-1", "recorded Put must have correct item")

		getCall := s.OnGet.LastCall(t)
		testkit.Equal(t, getCall.Id, "order-1", "recorded Get must have correct ID")
	})

	t.Run("fault injection overrides domain logic", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()
		s.OnPut.Faults(stub.ErrConflict, 1) // every call faults

		err := s.Put(t.Context(), stub.Item{ID: "new-item"})
		testkit.ErrorIs(t, err, stub.ErrConflict, "fault must override domain logic")

		// Item should NOT be stored since fault fired before domain logic.
		_, err = s.Get(t.Context(), "new-item")
		testkit.ErrorIs(t, err, stub.ErrNotFound, "faulted Put must not store item")
	})

	t.Run("concurrent access is safe", func(t *testing.T) {
		t.Parallel()
		s := stub.NewInMemoryStore()

		testkit.ConcurrentStress(t, 4, 10, func(g, i int) {
			id := testkit.SeededRand(t).IntN(100)
			item := stub.Item{ID: string(rune('A' + id))}
			_ = s.Put(t.Context(), item)
			_, _ = s.Get(t.Context(), item.ID)
		})
	})
}
