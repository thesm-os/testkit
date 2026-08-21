// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
)

// TestWitnessSpelling pins the stamp vocabulary a witness answers in: bare
// for a builtin, package-qualified for a source type, empty for a form no
// stamp ever spells.
func TestWitnessSpelling(t *testing.T) {
	t.Parallel()

	testkit.Equal(t, witnessSpelling(sdk.Builtin("int")), "int", "a builtin is its own spelling")
	testkit.Equal(t, witnessSpelling(sdk.External("example.com/x", "Score")),
		"example.com/x.Score", "a named type carries its package")
	testkit.Equal(t, witnessSpelling(nil), "", "and nothing else spells")
}

// TestSubstQ pins the substitution's two arms: a bound parameter name lands
// at its witness, everything else passes through.
func TestSubstQ(t *testing.T) {
	t.Parallel()

	b := &Bindings{witnessQ: map[string]string{"V": "int"}}
	testkit.Equal(t, b.substQ("V"), "int", "a parameter name lands at its witness")
	testkit.Equal(t, b.substQ("string"), "string", "a concrete spelling passes through")
}
