// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ref_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/ref"
)

var errClosed = errors.New("closed")

func TestBoundedCursor(t *testing.T) {
	t.Parallel()

	t.Run("Next yields each item once until exhausted", func(t *testing.T) {
		t.Parallel()
		c := ref.NewBoundedCursor([]string{"a", "b"}, errClosed)
		v, ok, _ := c.Next(t.Context())
		testkit.True(t, ok, "first")
		testkit.Equal(t, v, "a", "a yielded")
		v, ok, _ = c.Next(t.Context())
		testkit.True(t, ok, "second")
		testkit.Equal(t, v, "b", "b yielded")
		_, ok, _ = c.Next(t.Context())
		testkit.False(t, ok, "exhausted")
	})

	t.Run("Close is idempotent", func(t *testing.T) {
		t.Parallel()
		c := ref.NewBoundedCursor([]string{"a"}, errClosed)
		testkit.NoError(t, c.Close(t.Context()), "first close")
		testkit.NoError(t, c.Close(t.Context()), "second close")
		testkit.True(t, c.IsClosed(), "closed flag")
	})

	t.Run("Next after Close returns sentinel error", func(t *testing.T) {
		t.Parallel()
		c := ref.NewBoundedCursor([]string{"a"}, errClosed)
		_ = c.Close(t.Context())
		_, ok, err := c.Next(t.Context())
		testkit.False(t, ok, "no value after close")
		testkit.True(t, errors.Is(err, errClosed), "sentinel returned")
	})
}
