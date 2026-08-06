// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package sentinel generates the checks a package's error contract needs and
// nobody writes.
//
// A sentinel error is an API. Callers match on it with [errors.Is], read it in
// logs, and branch on it — so its message, its identity, and its behaviour
// under wrapping are as much a contract as any exported function's signature,
// and none of that is enforced by the compiler. Two sentinels that share a
// message are indistinguishable in a log. One that stops matching once wrapped
// is unusable at a boundary. A custom error type whose message drops a field it
// carries hides the one detail an operator needed. Each of those is a
// one-character mistake away at all times, and each is invisible until
// production.
//
// # What is read
//
// The directive sits at package scope, so what opts a package in and what the
// generator reads are different things. Two sets are collected:
//
//   - Exported package-level variables named `Err…`, which form the sentinel
//     set. The convention is load-bearing: a sentinel not named this way is not
//     found, which is the same rule every Go codebase already follows.
//   - Exported types declaring `Error() string`, which form the custom-error
//     set. Whether each also declares `Is` or `Unwrap` decides which further
//     checks it earns — a type without `Is` gets no self-consistency check
//     rather than a vacuous one.
//
// # What is emitted
//
// One test file per annotated package, holding an umbrella check over the
// sentinel set and one check per custom error type. Nothing production-side is
// generated: the author writes the errors, this asserts their invariants.
//
// # Prefix
//
// The prefix subtest asserts every sentinel's message begins with the package's
// name. `prefix=<value>` overrides what to expect, for a package whose errors
// are named for the subsystem rather than the directory. `prefix=off`
// suppresses the subtest, for a package whose errors are deliberately bare —
// and suppression has to be explicit, because a generator that silently skipped
// the check when it failed would be worse than not having it.
//
// # Cross-package non-overlap
//
// `sentinel-no-overlap-with` names another package whose sentinels must not
// satisfy [errors.Is] against this one's. Distinct values describing the same
// conditions — a `ErrNotFound` in two packages — are exactly where a
// copy-paste produces an accidental alias, and that is invisible within either
// package on its own. The directive repeats; each line unions.
package sentinel
