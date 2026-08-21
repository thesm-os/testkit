// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stamp

import "go.thesmos.sh/eidos/sdk"

// MetaRole holds the declared role, verbatim. Absent means the
// declaration filled none — an unroled field draws no pool.
//
//nolint:gochecknoglobals // meta key registration, immutable after init.
var MetaRole = sdk.EnsureKey("testkit.role", sdk.StringParser)

// RoleOf reads a declaration's stamped role, empty for the unroled — the
// one read path, so no consumer touches the key directly.
func RoleOf(bag *sdk.Bag) string {
	v, _ := MetaRole.Get(bag)
	return v
}
