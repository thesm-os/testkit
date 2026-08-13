// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"testing"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
)

// TestWitnessedPartner pins the nilable half of the guard substitution: an
// absent partner stays absent, a present one lands at the witnesses, and a
// non-generic interface passes its methods through untouched.
func TestWitnessedPartner(t *testing.T) {
	t.Parallel()

	testkit.True(t, witnessedPartner(nil, map[string]sdk.Ref{"V": sdk.Builtin("int")}) == nil,
		"no partner, nothing to rewrite")

	m := Method{Sig: &golang.Sig{
		Name:    "Put",
		Params:  []golang.Param{{Name: "v", Type: sdk.Builtin("V")}},
		Returns: []golang.Return{{Type: sdk.Builtin("error")}},
	}}
	testkit.True(t, witnessedPartner(&m, nil) == &m,
		"a nil binding map is the non-generic passthrough")

	w := witnessedPartner(&m, map[string]sdk.Ref{"V": sdk.Builtin("int")})
	got, isBuiltin := w.Sig.Params[0].Type.(*sdk.BuiltinRef)
	testkit.True(t, isBuiltin && got.Name == "int", "the parameter lands at the witness")
	orig := m.Sig.Params[0].Type.(*sdk.BuiltinRef)
	testkit.Equal(t, orig.Name, "V", "and the shared projection is untouched")
}
