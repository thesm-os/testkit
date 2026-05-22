// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package bmc is a bounded model checker: a depth-first search over
// a registered action set that exhaustively explores all reachable
// state-sequences up to the configured depth, checking every
// supplied invariant at every reachable state.
//
// BMC complements the property-engine random-search path: when the
// reachable state space is small (≤ a few thousand distinct states),
// BMC proves the absence of law violations within bounds rather
// than sampling for them. When a violation does exist, BMC returns
// the shortest action sequence that produces it.
//
// State-equivalence pruning is the difference between "intractable"
// and "feasible." When the consumer supplies a [Config.StateHash],
// two reached states with the same hash are treated as equivalent
// and the second is not re-explored. The hash is a consumer
// concern; refmap.MapStore.Snapshot() is the canonical
// fingerprint for CRUD-shaped interfaces.
package bmc
