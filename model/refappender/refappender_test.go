// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package refappender_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/model/refappender"
)

func TestMonotonicLog(t *testing.T) {
	t.Parallel()

	t.Run("Append returns monotonic gap-free offsets from 0", func(t *testing.T) {
		t.Parallel()
		l := refappender.NewMonotonicLog[string]()
		ctx := t.Context()
		o1, _ := l.Append(ctx, "a")
		o2, _ := l.Append(ctx, "b")
		o3, _ := l.Append(ctx, "c")
		testkit.Equal(t, o1, int64(0), "first offset")
		testkit.Equal(t, o2, int64(1), "second offset")
		testkit.Equal(t, o3, int64(2), "third offset")
	})

	t.Run("At returns entries by offset, false for out-of-range", func(t *testing.T) {
		t.Parallel()
		l := refappender.NewMonotonicLog[string]()
		_, _ = l.Append(t.Context(), "a")
		v, ok := l.At(0)
		testkit.True(t, ok, "in range")
		testkit.Equal(t, v, "a", "value")

		_, ok = l.At(-1)
		testkit.False(t, ok, "negative out of range")
		_, ok = l.At(99)
		testkit.False(t, ok, "above-range")
	})

	t.Run("Snapshot returns an independent copy", func(t *testing.T) {
		t.Parallel()
		l := refappender.NewMonotonicLog[string]()
		ctx := t.Context()
		_, _ = l.Append(ctx, "a")
		_, _ = l.Append(ctx, "b")
		snap := l.Snapshot()
		testkit.Equal(t, snap, []string{"a", "b"}, "snapshot contents")
		snap[0] = "mutated"
		again := l.Snapshot()
		testkit.Equal(t, again, []string{"a", "b"}, "internal state unchanged")
	})

	t.Run("Len tracks appends", func(t *testing.T) {
		t.Parallel()
		l := refappender.NewMonotonicLog[string]()
		testkit.Equal(t, l.Len(), 0, "empty")
		_, _ = l.Append(t.Context(), "x")
		testkit.Equal(t, l.Len(), 1, "one appended")
	})
}
