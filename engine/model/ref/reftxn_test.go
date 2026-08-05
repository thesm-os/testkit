// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ref_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/ref"
)

func TestSnapshotIsolation(t *testing.T) {
	t.Parallel()

	t.Run("Commit applies buffered writes atomically", func(t *testing.T) {
		t.Parallel()
		s := ref.NewSnapshotIsolation[string, int](errNotFound)
		tx, _ := s.Begin(t.Context())
		_ = tx.Put(t.Context(), "k", 42)
		_ = tx.Commit(t.Context())
		v, _ := s.Get(t.Context(), "k")
		testkit.Equal(t, v, 42, "commit visible")
	})

	t.Run("Rollback discards buffered writes", func(t *testing.T) {
		t.Parallel()
		s := ref.NewSnapshotIsolation[string, int](errNotFound)
		tx, _ := s.Begin(t.Context())
		_ = tx.Put(t.Context(), "k", 42)
		_ = tx.Rollback(t.Context())
		_, err := s.Get(t.Context(), "k")
		testkit.True(t, errors.Is(err, errNotFound), "rollback discarded")
	})

	t.Run("intra-tx reads see own writes", func(t *testing.T) {
		t.Parallel()
		s := ref.NewSnapshotIsolation[string, int](errNotFound)
		tx, _ := s.Begin(t.Context())
		_ = tx.Put(t.Context(), "k", 99)
		v, _ := tx.Get(t.Context(), "k")
		testkit.Equal(t, v, 99, "self write visible")
	})

	t.Run("snapshot isolation: concurrent commit invisible to active tx", func(t *testing.T) {
		t.Parallel()
		s := ref.NewSnapshotIsolation[string, int](errNotFound)
		ta, _ := s.Begin(t.Context())
		tb, _ := s.Begin(t.Context())
		_ = tb.Put(t.Context(), "k", 7)
		_ = tb.Commit(t.Context())
		// ta took its snapshot before tb committed.
		_, err := ta.Get(t.Context(), "k")
		testkit.True(t, errors.Is(err, errNotFound), "ta sees pre-commit snapshot")
	})

	t.Run("operations on closed tx error", func(t *testing.T) {
		t.Parallel()
		s := ref.NewSnapshotIsolation[string, int](errNotFound)
		tx, _ := s.Begin(t.Context())
		_ = tx.Commit(t.Context())
		err := tx.Put(t.Context(), "k", 1)
		testkit.True(t, errors.Is(err, ref.ErrTxClosed), "Put errors")
		_, err = tx.Get(t.Context(), "k")
		testkit.True(t, errors.Is(err, ref.ErrTxClosed), "Get errors")
		err = tx.Commit(t.Context())
		testkit.True(t, errors.Is(err, ref.ErrTxClosed), "double-commit errors")
		err = tx.Rollback(t.Context())
		testkit.True(t, errors.Is(err, ref.ErrTxClosed), "rollback after close errors")
	})

	// Read-your-own-writes is the buffer's reason to exist: a value written
	// inside the transaction must be visible to that transaction alone.
	t.Run("a transaction reads its own uncommitted write", func(t *testing.T) {
		t.Parallel()
		s := ref.NewSnapshotIsolation[string, int](errNotFound)
		tx, err := s.Begin(t.Context())
		testkit.NoError(t, err, "begin")

		testkit.NoError(t, tx.Put(t.Context(), "k", 7), "put")

		got, err := tx.Get(t.Context(), "k")
		testkit.NoError(t, err, "the buffer answers before the snapshot")
		testkit.Equal(t, got, 7, "the transaction sees its own write")

		_, err = s.Get(t.Context(), "k")
		testkit.ErrorIs(t, err, errNotFound, "the store sees nothing until commit")
	})

	// With nothing buffered, a read falls through to the snapshot taken at
	// Begin — the isolation the type is named for.
	t.Run("a transaction reads committed state from its snapshot", func(t *testing.T) {
		t.Parallel()
		s := ref.NewSnapshotIsolation[string, int](errNotFound)
		seed, err := s.Begin(t.Context())
		testkit.NoError(t, err, "begin")
		testkit.NoError(t, seed.Put(t.Context(), "k", 1), "put")
		testkit.NoError(t, seed.Commit(t.Context()), "commit")

		tx, err := s.Begin(t.Context())
		testkit.NoError(t, err, "begin")
		got, err := tx.Get(t.Context(), "k")
		testkit.NoError(t, err, "the snapshot answers when the buffer does not")
		testkit.Equal(t, got, 1, "committed value visible through the snapshot")
	})
}
