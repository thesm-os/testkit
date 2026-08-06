// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package priority is the enum-kind fixture whose zero value is a declared
// variant.
//
// It differs from [go.thesmos.sh/testkit/conformance/corpus/enum/status] in
// exactly one respect, and that is the point: Low is both the zero value and
// the first declaration, where status starts at iota+1. The generated
// zero-value check reads differently in each case — one asserts the zero is a
// variant, the other that it is not — and a fixture set where every enum makes
// the same choice cannot tell the two renderings apart.
//
// Four variants rather than three, so the count in the exhaustiveness check is
// not the same number as the neighbouring fixture's. A check that hard-coded
// one enum's arity would pass against the other by coincidence.
//
// No routing directive, and there cannot be one. The generated API declares
// methods on this type, and Go permits that only in the type's own package —
// an `out` sending it elsewhere produces a file naming an undefined type. The
// generated checks travel with it and take the external test package the
// _test.go suffix gives them.
package priority

// Priority is the enumerated type.
//
//testkit:enum
type Priority int

// The declared values, starting at zero.
const (
	Low Priority = iota
	Medium
	High
	Critical
)
