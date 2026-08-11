// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package suite generates a conformance harness for a Go interface.
//
// The harness is what an implementation is held to: `AssertMixedContract(t,
// factory)` runs every check the interface's signature and classifications
// imply, against implementations the caller supplies. Passing it is the claim
// that one implementation can stand in for another.
//
// # The unit is the method, not the classification
//
// A method's checks come from four sources, and only the third is a directive.
// The signature owes a smoke call, and — where a context is taken and an error
// returned — cancellation, deadline and nil-context checks, plus a check that
// an error carries the zero value. The detector adds what is specific to the
// shape. Each mixin and contract adds the direct form of its law. The typed
// extension point exists whether or not anything filled it.
//
// Counting only directives is what produces a file asserting four things about
// a three-method interface. The signature alone owes ten of the eleven it
// actually carries; one directive supplies the last.
//
// # One tier owns each classification
//
// [engine/model/law] implements seventy-one properties, and their names line up
// with eidos's classification vocabulary almost row for row. This generator
// implements none of them: where a law exists the classification is the model
// tier's, and the generated file's header says so (docs/adr/0018). What is left
// here is the signature-derived family, the shapes the law catalogue does not
// reach, and the classifications whose direct form is a fixed call sequence.
//
// The division is not about strength. It is about what a tier can state at all:
// a check for `cas` written where no stale version can be produced passes
// against every implementation, including a broken one.
//
// # Derivation first
//
// A consumer supplies a factory and nothing else. Every input the checks need
// is derived — the hit key, a miss key that differs from it by construction, a
// sample value per parameter type, and the seed, which calls the interface's own
// writer where it declares one.
//
// An option a consumer has to write is a derivation that has not been done. The
// three that remain exist for what derivation cannot reach: a seed for an
// interface with no writer, a check no classification implies, and dropping a
// generated check that is wrong for one subject.
//
// [engine/model/law]: https://go.thesmos.sh/testkit/engine/model/law
package suite
