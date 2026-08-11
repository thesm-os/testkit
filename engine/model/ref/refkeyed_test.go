// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ref_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/ref"
)

// TestKeyedStore walks the whole surface: the keyed put, the miss, the
// replace, the count, and the delete that succeeds on an absent key.
func TestKeyedStore(t *testing.T) {
	t.Parallel()

	miss := errors.New("reftest: not found")
	ctx := t.Context()

	t.Run("a put is observable and a miss reports the sentinel", func(t *testing.T) {
		t.Parallel()
		s := ref.NewKeyedStore[string, string](miss)

		_, err := s.Get(ctx, "k")
		testkit.ErrorIs(t, err, miss, "an empty store misses with the sentinel")

		testkit.NoError(t, s.Put(ctx, "k", "v"), "a put succeeds")
		got, err := s.Get(ctx, "k")
		testkit.NoError(t, err, "and is found under its key")
		testkit.Equal(t, got, "v", "holding what was put")

		testkit.NoError(t, s.Put(ctx, "k", "w"), "a second put replaces")
		got, _ = s.Get(ctx, "k")
		testkit.Equal(t, got, "w", "latest write wins")
	})

	t.Run("count follows puts and deletes", func(t *testing.T) {
		t.Parallel()
		s := ref.NewKeyedStore[string, int](miss)

		n, err := s.Count(ctx)
		testkit.NoError(t, err, "counting an empty store succeeds")
		testkit.Equal(t, n, 0, "at zero")

		testkit.NoError(t, s.Put(ctx, "a", 1), "first put")
		testkit.NoError(t, s.Put(ctx, "b", 2), "second put")
		n, _ = s.Count(ctx)
		testkit.Equal(t, n, 2, "two keys hold values")

		testkit.NoError(t, s.Delete(ctx, "a"), "a delete succeeds")
		n, _ = s.Count(ctx)
		testkit.Equal(t, n, 1, "and the count follows")

		_, err = s.Get(ctx, "a")
		testkit.ErrorIs(t, err, miss, "the deleted key misses")
	})

	t.Run("deleting an absent key succeeds", func(t *testing.T) {
		t.Parallel()
		s := ref.NewKeyedStore[string, string](miss)
		testkit.NoError(t, s.Delete(ctx, "never-put"),
			"the postcondition already holds")
	})
}
