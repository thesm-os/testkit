// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package clock provides deterministic time control for tests.
//
// [Clock] is the interface consumed by stubs and fault strategies.
// [RealClock] delegates to the standard library. [TestClock] allows
// manual time advancement with [TestClock.Advance] and deterministic
// synchronization via [TestClock.AwaitWaiters].
package clock
