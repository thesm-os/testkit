// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ref_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/ref"
)

var errReject = errors.New("rejected")

// Three-state lifecycle: created → active → closed.
const (
	created = "created"
	active  = "active"
	closed  = "closed"
)

// Transitions: "activate" moves created→active; "close" moves active→closed.
func guard(from, m string) (string, bool) {
	switch {
	case from == created && m == "activate":
		return active, true
	case from == active && m == "close":
		return closed, true
	}
	return from, false
}

func TestCompliance(t *testing.T) {
	t.Parallel()

	t.Run("untouched key reports the initial state", func(t *testing.T) {
		t.Parallel()
		c := ref.NewGuardedStates[string, string, string](created, guard, errReject)
		s, _ := c.State(t.Context(), "x")
		testkit.Equal(t, s, created, "initial")
	})

	t.Run("valid transition advances state", func(t *testing.T) {
		t.Parallel()
		c := ref.NewGuardedStates[string, string, string](created, guard, errReject)
		testkit.NoError(t, c.Apply(t.Context(), "x", "activate"), "activate")
		s, _ := c.State(t.Context(), "x")
		testkit.Equal(t, s, active, "advanced")
	})

	t.Run("invalid transition errors and preserves state", func(t *testing.T) {
		t.Parallel()
		c := ref.NewGuardedStates[string, string, string](created, guard, errReject)
		err := c.Apply(t.Context(), "x", "close") // close from created is invalid
		testkit.True(t, errors.Is(err, errReject), "rejected")
		s, _ := c.State(t.Context(), "x")
		testkit.Equal(t, s, created, "unchanged")
	})

	t.Run("full lifecycle activates then closes", func(t *testing.T) {
		t.Parallel()
		c := ref.NewGuardedStates[string, string, string](created, guard, errReject)
		_ = c.Apply(t.Context(), "x", "activate")
		testkit.NoError(t, c.Apply(t.Context(), "x", "close"), "close")
		s, _ := c.State(t.Context(), "x")
		testkit.Equal(t, s, closed, "closed")
	})
}
