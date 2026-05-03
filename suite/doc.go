// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package suite provides the runtime types and conformance primitives
// for generated AssertXxxContract test harnesses.
//
// Each method shape (Reader, Writer, Deleter, Aggregator, Lifecycle,
// Pure, Predicate, Stream, Cross) has a typed Context and a set of
// assertion primitives. The generated suite code constructs Contexts
// and dispatches consumer-provided primitives through them.
//
// Fault injection hooks ([WithReaderFaults], etc.) wrap Bindings to
// apply [testkit.Fault] strategies before invocation.
package suite
