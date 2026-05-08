// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package defaultstest is the sibling test package that holds
// hand-written defaults factories paired with the corresponding
// types in defaults. The builder generator emits its output here so
// the factory and the generated NewRequest/Build helpers live in
// the same package — solving the "where does the factory go?"
// chicken-and-egg.
package defaultstest

import "go.thesmos.sh/testkit/generator/testdata/defaults"

// RequestDefaults returns the canonical seed value the generated
// NewRequest() builder uses. The builder generator detects this
// function via the `<Type>Defaults() <Type>` shape and wires it
// in automatically.
func RequestDefaults() defaults.Request {
	return defaults.Request{
		RunID: "test-run-id",
		Token: 42,
		Data:  []byte("test-data"),
	}
}
