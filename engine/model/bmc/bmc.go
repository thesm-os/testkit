// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bmc

import (
	"errors"
	"fmt"
)

// Action is one step the BMC engine may take. The engine applies
// Action.Apply to the current state to compute the next state.
// Apply returning an error skips the action without violating any
// invariant (typical for actions whose precondition isn't met in
// the current state — e.g., Dec on an empty counter).
type Action[T any] struct {
	// Name appears in counterexample sequences.
	Name string

	// Apply produces the successor state. Returning a non-nil
	// error skips the action (no successor explored). Apply must
	// be deterministic: same input state must always produce the
	// same output state for the same Action.
	Apply func(T) (T, error)
}

// Invariant is a deterministic predicate checked at every reached
// state. Returning a non-nil error halts the search and surfaces
// the action sequence as a counterexample.
type Invariant[T any] struct {
	// Name appears in [Outcome.ViolatedInvariant].
	Name string

	// Check returns nil when the invariant holds for state, or a
	// diagnostic error when it does not. Check must be
	// deterministic and side-effect-free.
	Check func(T) error
}

// Config bounds the exploration.
type Config[T any] struct {
	// Depth caps the longest action sequence the engine explores.
	// A depth of 0 checks invariants on the initial state only.
	// Negative depths are treated as 0.
	Depth int

	// Commands caps the number of distinct actions the engine
	// considers at each step (breadth bound). When Commands > 0
	// and Commands < len(actions), only the first Commands entries
	// in the actions slice are explored. Zero or larger-than-len
	// disables the cap. Used to shrink the state space when the
	// consumer wants BMC over a subset of the action vocabulary.
	Commands int

	// StateHash returns a string fingerprint of state for
	// equivalence pruning. When two reached states hash equal, the
	// engine skips re-exploring the second. nil disables pruning
	// (every reach is fresh) — useful when reasoning about hash
	// completeness is harder than the exploration cost.
	StateHash func(T) string
}

// Outcome is the BMC result.
type Outcome[T any] struct {
	// Explored is the count of distinct states visited (after
	// pruning). Used to validate the bound was reasonable and to
	// quantify state-space coverage.
	Explored int

	// Pruned is the count of states the engine elided via
	// StateHash equivalence. Zero when StateHash is nil.
	Pruned int

	// Counterexample is the action-name sequence that produced
	// the violation, in order. Empty when no violation was found.
	Counterexample []string

	// ViolatedInvariant is the Name of the invariant that fired.
	// Empty when no violation was found.
	ViolatedInvariant string

	// Reason is the violating invariant's diagnostic message.
	Reason string

	// FailingState is the state at the point of the violation —
	// the result of applying the final action in Counterexample.
	// Zero value when no violation was found.
	FailingState T
}

// Violated reports whether the run found a counterexample.
func (o Outcome[T]) Violated() bool {
	return o.ViolatedInvariant != ""
}

// Run performs the DFS. The initial state is checked against every
// invariant before any action is applied; if the initial state
// already violates an invariant, the returned Outcome has an empty
// Counterexample naming the violation.
//
// The engine is exhaustive within Depth: when it returns without a
// counterexample, no action sequence of length ≤ Depth from the
// initial state violates any supplied invariant — a proof of
// safety within bounds.
func Run[T any](initial T, actions []Action[T], invariants []Invariant[T], cfg Config[T]) Outcome[T] {
	if cfg.Depth < 0 {
		cfg.Depth = 0
	}
	if cfg.Commands > 0 && cfg.Commands < len(actions) {
		actions = actions[:cfg.Commands]
	}
	visited := map[string]struct{}{}
	pruned := 0

	out := Outcome[T]{}
	out.Explored = 1
	if cfg.StateHash != nil {
		visited[cfg.StateHash(initial)] = struct{}{}
	}

	if name, reason, ok := checkInvariants(initial, invariants); ok {
		out.ViolatedInvariant = name
		out.Reason = reason
		out.FailingState = initial
		return out
	}

	var trail []string
	explore(initial, actions, invariants, cfg, 0, &trail, visited, &out, &pruned)
	out.Pruned = pruned
	return out
}

// explore is the recursive DFS body. It writes the counterexample into out
// when an invariant fires; the whole stack then unwinds through the
// violation check at the top of the action loop, so no deeper state is
// visited once a counterexample exists.
func explore[T any](
	state T,
	actions []Action[T],
	invariants []Invariant[T],
	cfg Config[T],
	depth int,
	trail *[]string,
	visited map[string]struct{},
	out *Outcome[T],
	pruned *int,
) {
	if depth >= cfg.Depth {
		return
	}

	for _, a := range actions {
		if out.Violated() {
			return
		}
		next, err := a.Apply(state)
		if err != nil {
			continue // precondition unmet; skip
		}

		if cfg.StateHash != nil {
			h := cfg.StateHash(next)
			if _, seen := visited[h]; seen {
				*pruned++
				continue
			}
			visited[h] = struct{}{}
		}
		out.Explored++

		*trail = append(*trail, a.Name)
		if name, reason, fired := checkInvariants(next, invariants); fired {
			out.ViolatedInvariant = name
			out.Reason = reason
			out.Counterexample = append([]string(nil), *trail...)
			out.FailingState = next
			return
		}
		explore(next, actions, invariants, cfg, depth+1, trail, visited, out, pruned)
		*trail = (*trail)[:len(*trail)-1]
	}
}

func checkInvariants[T any](state T, invariants []Invariant[T]) (name, reason string, fired bool) {
	for _, inv := range invariants {
		if err := inv.Check(state); err != nil {
			return inv.Name, err.Error(), true
		}
	}
	return "", "", false
}

// ErrPreconditionUnmet is the canonical error consumers may return
// from an Action's Apply when the action does not apply to the
// current state. Other errors are equally accepted by the engine;
// this is a convenience for the common case.
var ErrPreconditionUnmet = errors.New("bmc: action precondition unmet")

// Errorf builds a diagnostic for an Invariant.Check return.
// Inverse of fmt.Errorf for callers that want a stable prefix in
// the Outcome.Reason.
func Errorf(format string, a ...any) error {
	return fmt.Errorf("invariant: "+format, a...)
}
