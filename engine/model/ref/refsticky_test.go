// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ref_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/ref"
)

type stickyDoc struct{ ID, Body string }

// TestStickyStore pins the refinement the sticky claim buys: the first
// resolution persists through overwrites, a miss pins nothing, and a delete
// unpins so the key's next resolution starts afresh.
func TestStickyStore(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	miss := errors.New("sticky test: not found")
	s := ref.NewStickyStore(func(d stickyDoc) string { return d.ID }, miss)

	_, err := s.Get(ctx, "k")
	testkit.ErrorIs(t, err, miss, "a miss reports the sentinel")

	testkit.NoError(t, s.Put(ctx, stickyDoc{ID: "k", Body: "first"}), "the write after a miss")
	got, err := s.Get(ctx, "k")
	testkit.NoError(t, err, "and now the key resolves")
	testkit.Equal(t, got.Body, "first", "to what the miss left room for — a miss pinned nothing")

	testkit.NoError(t, s.Put(ctx, stickyDoc{ID: "k", Body: "second"}), "an overwrite lands")
	got, _ = s.Get(ctx, "k")
	testkit.Equal(t, got.Body, "first", "but the resolution sticks")

	held, _ := s.Values(ctx)
	testkit.Equal(t, held[0].Body, "second",
		"while the store itself holds the latest write — pinning is Get's business")

	testkit.NoError(t, s.Delete(ctx, "k"), "the delete")
	_, err = s.Get(ctx, "k")
	testkit.ErrorIs(t, err, miss, "ends the key's story rather than serving the pin")

	testkit.NoError(t, s.Put(ctx, stickyDoc{ID: "k", Body: "third"}), "a re-add")
	got, _ = s.Get(ctx, "k")
	testkit.Equal(t, got.Body, "third", "resolves afresh")
}
