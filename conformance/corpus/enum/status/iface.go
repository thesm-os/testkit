// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package status is the enum-kind fixture whose zero value is not a declared
// variant.
//
// The production surface — String, ParseStatus, MarshalJSON, UnmarshalJSON and
// the parse-error sentinel — is generated rather than written here, so the
// fixture declares only the type and its variants. Writing a String by hand
// would collide with the generated one, which is why none appears below.
//
// Draft is deliberately non-zero. That makes an unset Status invalid rather
// than silently meaning the first variant, and it is what lets the generated
// checks assert something about the zero value at all —
// [go.thesmos.sh/testkit/conformance/corpus/enum/priority] makes the opposite
// choice, and the two assertions read differently.
//
// The sequence has no gap: a gap would be indistinguishable from an
// out-of-range value in the fallback check, and the boundary the check probes
// is the one past the last declared variant.
//
// No routing directive, and there cannot be one. The generated API declares
// methods on this type, and Go permits that only in the type's own package —
// an `out` sending it elsewhere produces a file naming an undefined type. The
// generated checks travel with it and take the external test package the
// _test.go suffix gives them.
package status

// Status is the enumerated type.
//
//testkit:enum
type Status int

// The declared values.
const (
	Draft Status = iota + 1
	Published
	Archived
)
