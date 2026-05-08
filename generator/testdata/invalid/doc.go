// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package invalid holds deliberately-malformed fixtures used to
// verify generator predicates correctly reject (or skip) bad inputs:
//
//   - stringenum.go — string-typed Tag; the enum generator rejects
//                     non-integer underlying kinds.
//   - wrongsigs.go  — WrongSig has methods that look like
//                     Stringer/Parse/marshalers but with wrong
//                     signatures. Predicate detection must return
//                     false for all of them.
//
// Deliberately has no `go:generate` directives — these are inputs
// to predicate tests, not end-to-end generation.
package invalid
