// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package priority is the bare-iota half of the enum-kind corpus: a typed
// constant block carrying no String, Parse, or Marshal methods.
//
// The generator branches on whether those methods exist. With them it appends
// round-trip subtests and a wantStr field to the generated table; without them
// it emits only the always-emitted subtests and a table that has no such
// field. Both branches produce a different file, so a corpus covering only the
// method-bearing case leaves half the generator unexercised — which is what
// [go.thesmos.sh/testkit/conformance/corpus/enum/status] covers and this does
// not.
//
// The zero value is deliberately a declared constant here, where status starts
// at iota+1. The generated "zero value is X" assertion reads differently in
// each case, and a fixture set where every enum makes the same choice cannot
// tell the two renderings apart.
package priority

// Priority is the enumerated type. It has no methods at all.
type Priority int

// The declared values, starting at zero so Low is both the zero value and the
// first declaration.
const (
	Low Priority = iota
	Medium
	High
	Critical
)
