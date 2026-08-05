// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ref_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/ref"
)

func TestAtLeastOnce(t *testing.T) {
	t.Parallel()

	t.Run("every subscriber receives every publish", func(t *testing.T) {
		t.Parallel()
		b := ref.NewAtLeastOnce[string]()
		a, _ := b.Subscribe(t.Context())
		c, _ := b.Subscribe(t.Context())
		_ = b.Publish(t.Context(), "hi")
		_ = b.Publish(t.Context(), "yo")
		ga, _ := b.Drain(t.Context(), a)
		gc, _ := b.Drain(t.Context(), c)
		testkit.Equal(t, ga, []string{"hi", "yo"}, "a got both")
		testkit.Equal(t, gc, []string{"hi", "yo"}, "c got both")
	})

	t.Run("redelivery via re-publish models at-least-once duplicates", func(t *testing.T) {
		t.Parallel()
		b := ref.NewAtLeastOnce[string]()
		a, _ := b.Subscribe(t.Context())
		_ = b.Publish(t.Context(), "x")
		_ = b.Publish(t.Context(), "x") // redelivery
		got, _ := b.Drain(t.Context(), a)
		testkit.Equal(t, got, []string{"x", "x"}, "duplicate observed")
	})
}

func TestAtMostOnce(t *testing.T) {
	t.Parallel()

	t.Run("publishes within capacity reach the subscriber", func(t *testing.T) {
		t.Parallel()
		b := ref.NewAtMostOnce[string](2)
		a, _ := b.Subscribe(t.Context())
		_ = b.Publish(t.Context(), "x")
		_ = b.Publish(t.Context(), "y")
		got, dropped, _ := b.Drain(t.Context(), a)
		testkit.Equal(t, got, []string{"x", "y"}, "both delivered")
		testkit.Equal(t, dropped, 0, "none dropped")
	})

	t.Run("publishes beyond capacity drop", func(t *testing.T) {
		t.Parallel()
		b := ref.NewAtMostOnce[string](1)
		a, _ := b.Subscribe(t.Context())
		_ = b.Publish(t.Context(), "x")
		_ = b.Publish(t.Context(), "y") // over capacity
		got, dropped, _ := b.Drain(t.Context(), a)
		testkit.Equal(t, got, []string{"x"}, "first delivered")
		testkit.Equal(t, dropped, 1, "second dropped")
	})

	t.Run("unlimited capacity buffers everything", func(t *testing.T) {
		t.Parallel()
		b := ref.NewAtMostOnce[int](-1)
		a, _ := b.Subscribe(t.Context())
		for i := range 100 {
			_ = b.Publish(t.Context(), i)
		}
		got, dropped, _ := b.Drain(t.Context(), a)
		testkit.Equal(t, len(got), 100, "all buffered")
		testkit.Equal(t, dropped, 0, "none dropped")
	})
}

func TestExactlyOnce(t *testing.T) {
	t.Parallel()

	t.Run("first publish delivers", func(t *testing.T) {
		t.Parallel()
		b := ref.NewExactlyOnce[string]()
		a, _ := b.Subscribe(t.Context())
		_, _ = b.Publish(t.Context(), "x")
		got, _ := b.Drain(t.Context(), a)
		testkit.Equal(t, got, []string{"x"}, "delivered")
	})

	t.Run("Replay of seen ID is dropped per subscriber", func(t *testing.T) {
		t.Parallel()
		b := ref.NewExactlyOnce[string]()
		a, _ := b.Subscribe(t.Context())
		id, _ := b.Publish(t.Context(), "x")
		_ = b.Replay(t.Context(), id, "x")
		got, _ := b.Drain(t.Context(), a)
		testkit.Equal(t, got, []string{"x"}, "replay deduped")
	})

	t.Run("late subscriber sees only post-subscribe publishes", func(t *testing.T) {
		t.Parallel()
		b := ref.NewExactlyOnce[string]()
		_, _ = b.Publish(t.Context(), "missed")
		late, _ := b.Subscribe(t.Context())
		_, _ = b.Publish(t.Context(), "caught")
		got, _ := b.Drain(t.Context(), late)
		testkit.Equal(t, got, []string{"caught"}, "late only sees post-subscribe")
	})
}
