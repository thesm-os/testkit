// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ref_test

import (
	"context"
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/ref"
)

type casEntry struct {
	ID      string
	Version int
}

var errMismatch = errors.New("version mismatch")

func newCell() *ref.AtomicCell[casEntry, int] {
	return ref.NewAtomicCell(
		func(e casEntry) int { return e.Version },
		func(v int) int { return v + 1 },
		errMismatch,
	)
}

func TestAtomicCell(t *testing.T) {
	t.Parallel()

	t.Run("Get on empty cell returns false", func(t *testing.T) {
		t.Parallel()
		c := newCell()
		_, _, ok := c.Get(t.Context())
		testkit.False(t, ok, "empty cell")
	})

	t.Run("first write succeeds regardless of version", func(t *testing.T) {
		t.Parallel()
		c := newCell()
		testkit.NoError(t, c.CompareAndSwap(t.Context(), casEntry{ID: "x", Version: 0}), "first")
		_, ver, ok := c.Get(t.Context())
		testkit.True(t, ok, "present after first write")
		testkit.Equal(t, ver, 1, "version advanced after first write")
	})

	t.Run("matching version updates", func(t *testing.T) {
		t.Parallel()
		c := newCell()
		_ = c.CompareAndSwap(t.Context(), casEntry{ID: "x", Version: 0})
		// After first write, stored version is 1. Updater must supply version=1.
		testkit.NoError(t, c.CompareAndSwap(t.Context(), casEntry{ID: "x", Version: 1}), "match")
		_, ver, _ := c.Get(t.Context())
		testkit.Equal(t, ver, 2, "version advanced")
	})

	t.Run("mismatched version errors", func(t *testing.T) {
		t.Parallel()
		c := newCell()
		_ = c.CompareAndSwap(t.Context(), casEntry{ID: "x", Version: 0})
		err := c.CompareAndSwap(t.Context(), casEntry{ID: "x", Version: 99})
		testkit.True(t, errors.Is(err, errMismatch), "mismatch error")
	})
}

// The fixture-faced cell: the version rides inside the value, the cell
// advances by one per accepted write, and both sentinels report by presence.
func TestVersionedCell(t *testing.T) {
	t.Parallel()

	type doc struct {
		Body    string
		Version int64
	}
	mismatch := errors.New("stale")
	empty := errors.New("empty")
	fresh := func() *ref.VersionedCell[doc] {
		return ref.NewVersionedCell(func(d doc) int64 { return d.Version }, mismatch, empty)
	}

	t.Run("an unwritten cell reads empty", func(t *testing.T) {
		t.Parallel()
		_, err := fresh().Get(t.Context())
		testkit.ErrorIs(t, err, empty, "nothing written, nothing read")
	})

	t.Run("a matching version wins and advances the cell", func(t *testing.T) {
		t.Parallel()
		c := fresh()
		testkit.NoError(t, c.Put(t.Context(), doc{Body: "a", Version: 0}), "zero matches a fresh cell")
		testkit.ErrorIs(t, c.Put(t.Context(), doc{Body: "b", Version: 0}), mismatch,
			"the cell advanced, so the same version is stale")
		testkit.NoError(t, c.Put(t.Context(), doc{Body: "b", Version: 1}), "the next version wins")

		got, err := c.Get(t.Context())
		testkit.NoError(t, err, "a written cell reads")
		testkit.Equal(t, got, doc{Body: "b", Version: 1}, "verbatim — the writer's own field")
	})

	t.Run("a cancelled context refuses both operations", func(t *testing.T) {
		t.Parallel()
		c := fresh()
		cancelled, cancel := context.WithCancel(t.Context())
		cancel()
		testkit.ErrorIs(t, c.Put(cancelled, doc{}), context.Canceled, "the write never lands")
		_, err := c.Get(cancelled)
		testkit.ErrorIs(t, err, context.Canceled, "the read never answers")
	})
}
