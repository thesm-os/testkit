// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package shrinker reduces a failing test trace to its essential
// signal. Three independent reducers compose:
//
//   - Causal: given a sequence of [Step]s annotated with reads
//     and writes, return only the steps in the failing action's
//     causal hull (the transitive read-of-writes closure). Often
//     reduces a 50-action sequence to a handful.
//   - Witness: format a failed sequence plus its diagnostic
//     message as a one-line failure predicate suitable for
//     `<artifactDir>/witness-<seed>.txt`.
//   - Race: given a goroutine-tagged event sequence that triggered
//     -race, identify the minimal pair of conflicting operations
//     and/or shrink the schedule via a consumer-supplied probe.
//
// Each reducer is independently usable; the model runner composes
// them on failure to produce a minimal artifact set.
package shrinker
