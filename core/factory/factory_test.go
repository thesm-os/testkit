// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package factory_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/factory"
)

func TestNewNamed(t *testing.T) {
	t.Parallel()

	t.Run("constructs with valid name and factory", func(t *testing.T) {
		t.Parallel()
		n := factory.NewNamed("v1", func() int { return 42 })
		testkit.Equal(t, n.Name(), "v1", "Name returns the constructor argument")
		testkit.Equal(t, n.Construct(), 42, "Construct returns the factory's value")
	})

	t.Run("panics on empty name", func(t *testing.T) {
		t.Parallel()
		recovered := testkit.Panics(t, func() {
			factory.NewNamed("", func() int { return 0 })
		}, "empty name must panic at construction")
		testkit.Assert(t, asString(recovered)).Contains("name is empty", "diagnostic")
	})

	t.Run("panics on nil factory", func(t *testing.T) {
		t.Parallel()
		recovered := testkit.Panics(t, func() {
			factory.NewNamed[int]("v1", nil)
		}, "nil factory must panic at construction")
		testkit.Assert(t, asString(recovered)).
			Contains("factory closure is nil", "diagnostic").
			Contains(`name="v1"`, "names the offending Named")
	})
}

func TestNamedConstruct(t *testing.T) {
	t.Parallel()

	t.Run("invokes the closure each call", func(t *testing.T) {
		t.Parallel()
		calls := 0
		n := factory.NewNamed("counter", func() int { calls++; return calls })
		_ = n.Construct()
		_ = n.Construct()
		_ = n.Construct()
		testkit.Equal(t, calls, 3, "factory invoked once per Construct")
	})

	t.Run("produces isolated values", func(t *testing.T) {
		t.Parallel()
		// The Named contract requires the factory to produce
		// isolated values across calls. A factory returning the
		// same backing array would fail the runner's per-iteration
		// isolation guarantee.
		n := factory.NewNamed("isolated", func() []int { return []int{1, 2, 3} })
		a := n.Construct()
		b := n.Construct()
		a[0] = 99
		testkit.Equal(t, b[0], 1, "factory must produce isolated slices")
	})
}

// asString renders an arbitrary panic value as a string for
// substring assertions. testkit.Panics returns `recover()`'s any-typed
// payload; panics raised with a string message round-trip cleanly.
func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if e, ok := v.(error); ok {
		return e.Error()
	}
	return ""
}
