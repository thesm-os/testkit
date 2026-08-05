// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ref_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/ref"
)

func TestCoalescer(t *testing.T) {
	t.Parallel()

	t.Run("first call invokes compute, second returns cached", func(t *testing.T) {
		t.Parallel()
		c := ref.NewCoalescer[string, int]()
		calls := 0
		v1, err := c.Do(t.Context(), "k", func() int { calls++; return 42 })
		testkit.NoError(t, err, "first")
		testkit.Equal(t, v1, 42, "computed")
		testkit.Equal(t, calls, 1, "fn called once")

		v2, _ := c.Do(t.Context(), "k", func() int { calls++; return 99 })
		testkit.Equal(t, v2, 42, "cached")
		testkit.Equal(t, calls, 1, "fn not re-invoked")
	})

	t.Run("Calls reports per-key invocation count", func(t *testing.T) {
		t.Parallel()
		c := ref.NewCoalescer[string, int]()
		_, _ = c.Do(t.Context(), "a", func() int { return 1 })
		_, _ = c.Do(t.Context(), "b", func() int { return 2 })
		_, _ = c.Do(t.Context(), "a", func() int { return 99 })
		testkit.Equal(t, c.Calls("a"), 1, "a coalesced")
		testkit.Equal(t, c.Calls("b"), 1, "b computed once")
		testkit.Equal(t, c.Calls("missing"), 0, "untouched key")
	})
}
