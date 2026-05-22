// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package reflease_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/model/reflease"
)

var (
	errHeld = errors.New("held")
	errFree = errors.New("free")
)

func TestTracker(t *testing.T) {
	t.Parallel()

	t.Run("Acquire on free key succeeds", func(t *testing.T) {
		t.Parallel()
		tr := reflease.NewTracker[string](errHeld, errFree)
		testkit.NoError(t, tr.Acquire(t.Context(), "k"), "first acquire")
		testkit.True(t, tr.IsHeld("k"), "held")
	})

	t.Run("double acquire errors", func(t *testing.T) {
		t.Parallel()
		tr := reflease.NewTracker[string](errHeld, errFree)
		_ = tr.Acquire(t.Context(), "k")
		err := tr.Acquire(t.Context(), "k")
		testkit.True(t, errors.Is(err, errHeld), "double acquire returns held")
	})

	t.Run("Release of held lease frees it", func(t *testing.T) {
		t.Parallel()
		tr := reflease.NewTracker[string](errHeld, errFree)
		_ = tr.Acquire(t.Context(), "k")
		testkit.NoError(t, tr.Release(t.Context(), "k"), "release")
		testkit.False(t, tr.IsHeld("k"), "freed")
	})

	t.Run("Release of unheld lease errors", func(t *testing.T) {
		t.Parallel()
		tr := reflease.NewTracker[string](errHeld, errFree)
		err := tr.Release(t.Context(), "k")
		testkit.True(t, errors.Is(err, errFree), "release-of-unheld returns free")
	})
}
