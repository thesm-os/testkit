// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package timeaware provides law-shaped checkers for interfaces
// that depend on a clock: TTL expiry, deadline propagation, and
// scheduled-task firing. Each checker drives the SUT through a
// scripted sequence of operations and clock advances, then asserts
// the post-advance observable state.
//
// The package also ships a [Barrier] primitive used by the model
// runner's concurrent path to synchronize clock advancement with
// in-flight operations: operations enter via [Barrier.Op] and
// release on completion; [Barrier.Advance] blocks until every
// in-flight op releases before applying the advance.
package timeaware
