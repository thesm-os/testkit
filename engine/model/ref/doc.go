// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package ref holds the reference implementations the model runner tests a
// subject against.
//
// Each type is a deliberately simple, obviously-correct in-memory model of one
// primitive — an atomic cell, an append-only chain, a lease tracker, a
// snapshot-isolated store. Differential testing runs the subject and the
// reference through the same action sequence and compares observations, so a
// reference is only useful while it stays simple enough to be read and
// believed without its own test suite.
//
// Types are named for what they model rather than for the law they serve, so a
// caller reads ref.AtomicCell and ref.SnapshotIsolation rather than a package
// prefix that repeats the import path.
//
// Concurrency: every reference guards its state with a mutex and is safe for
// concurrent use, because the concurrent-stress and linearizability checkers
// drive them from many goroutines. None of them are allocation-tuned; they are
// oracles, not production code.
package ref
