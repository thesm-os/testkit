// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package fault provides the fault-injection strategies test doubles use to
// fail on demand.
//
// A [Fault] decides, per call, whether to return an error. The strategies
// compose: [NewCountedFault] fails every Nth call, [NewWindowedFault] fails
// until a deadline, [NewPredicateFault] fails when the call matches, and [And]
// and [Or] combine them. Composition is how a test expresses "fail the third
// write, but only while the circuit is open" without writing a bespoke double.
//
// Determinism: [NewProbabilityFault] draws from a caller-supplied source, so a
// seeded run reproduces exactly. Time-based strategies read a caller-supplied
// clock rather than the wall clock for the same reason.
//
// Concurrency: strategies carry mutable call counters and are guarded
// internally, so one instance may back a stub shared across goroutines.
package fault
