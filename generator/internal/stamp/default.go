// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stamp

import "go.thesmos.sh/eidos/sdk"

// MetaDefault holds the field's default as Go source. Absent, or the empty
// string, means the field declared none — which is distinct from a default of
// zero, and the reason the stamp is not a typed value.
//
//nolint:gochecknoglobals // meta key registration, immutable after init.
var MetaDefault = sdk.EnsureKey("testkit.default", sdk.StringParser)

// MetaDefaultPkg holds the import path a qualified default resolved to, empty
// for a plain literal. Two keys rather than one string because a rendered file
// has to register the import, which only a reference can carry.
//
//nolint:gochecknoglobals // meta key registration, immutable after init.
var MetaDefaultPkg = sdk.EnsureKey("testkit.default.pkg", sdk.StringParser)

// DefaultOf returns the field's declared default as Go source, or empty when
// it declared none.
func DefaultOf(bag *sdk.Bag) string {
	out, _ := MetaDefault.Get(bag)
	return out
}

// DefaultPackage returns the import path a qualified default resolved to,
// empty when the default is a plain literal.
func DefaultPackage(bag *sdk.Bag) string {
	out, _ := MetaDefaultPkg.Get(bag)
	return out
}
