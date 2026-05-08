// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package interfaces holds interface-method-signature variations
// beyond the canonical Store in basic. Stub, suite, and bench
// consume types from here to cover the long tail of method shapes:
//
//   - variadic methods
//   - methods returning iterators (iter.Seq, iter.Seq2)
//   - methods without a context.Context parameter
//   - methods without an error return
//   - methods with named returns
//   - methods with multiple non-error returns
//   - methods with no parameters
//   - lifecycle methods (Close, Init)
//   - mutator vs reader vs writer shape variants
//
// Each file declares one or two interfaces focused on a single
// shape; tests scope by interface name (`stub VariadicFinder`,
// `suite IterScanner`, etc.).
package interfaces
