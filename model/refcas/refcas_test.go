// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package refcas_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/model/refcas"
)

type entry struct {
	ID      string
	Version int
}

var errMismatch = errors.New("version mismatch")

func newCell() *refcas.AtomicCell[entry, int] {
	return refcas.NewAtomicCell(
		func(e entry) int { return e.Version },
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
		testkit.NoError(t, c.CompareAndSwap(t.Context(), entry{ID: "x", Version: 0}), "first")
		_, ver, ok := c.Get(t.Context())
		testkit.True(t, ok, "present after first write")
		testkit.Equal(t, ver, 1, "version advanced after first write")
	})

	t.Run("matching version updates", func(t *testing.T) {
		t.Parallel()
		c := newCell()
		_ = c.CompareAndSwap(t.Context(), entry{ID: "x", Version: 0})
		// After first write, stored version is 1. Updater must supply version=1.
		testkit.NoError(t, c.CompareAndSwap(t.Context(), entry{ID: "x", Version: 1}), "match")
		_, ver, _ := c.Get(t.Context())
		testkit.Equal(t, ver, 2, "version advanced")
	})

	t.Run("mismatched version errors", func(t *testing.T) {
		t.Parallel()
		c := newCell()
		_ = c.CompareAndSwap(t.Context(), entry{ID: "x", Version: 0})
		err := c.CompareAndSwap(t.Context(), entry{ID: "x", Version: 99})
		testkit.True(t, errors.Is(err, errMismatch), "mismatch error")
	})
}
