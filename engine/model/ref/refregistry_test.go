// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ref_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/ref"
)

var errDup = errors.New("duplicate")

func TestBootOnly(t *testing.T) {
	t.Parallel()

	t.Run("first Register stores the value", func(t *testing.T) {
		t.Parallel()
		b := ref.NewBootOnlyRegistry[string, int](errDup)
		testkit.NoError(t, b.Register(t.Context(), "k", 1), "first")
		v, ok := b.Lookup(t.Context(), "k")
		testkit.True(t, ok, "present")
		testkit.Equal(t, v, 1, "value")
	})

	t.Run("re-Register returns duplicate error and preserves the original", func(t *testing.T) {
		t.Parallel()
		b := ref.NewBootOnlyRegistry[string, int](errDup)
		_ = b.Register(t.Context(), "k", 1)
		err := b.Register(t.Context(), "k", 99)
		testkit.True(t, errors.Is(err, errDup), "duplicate error")
		v, _ := b.Lookup(t.Context(), "k")
		testkit.Equal(t, v, 1, "original preserved")
	})

	t.Run("Len tracks registrations", func(t *testing.T) {
		t.Parallel()
		b := ref.NewBootOnlyRegistry[string, int](errDup)
		_ = b.Register(t.Context(), "a", 1)
		_ = b.Register(t.Context(), "b", 2)
		testkit.Equal(t, b.Len(), 2, "two entries")
	})
}
