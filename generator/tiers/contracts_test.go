// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/tiers"
)

// TestContractStoresAreInternallyCoherent holds the two contract tables to
// each other: every store row's roles carry ops, every op's contract carries
// a store, and the type-argument role is one the ops actually name — a row
// broken any of those ways derives an oracle whose constructor or adapter
// fails to compile in whichever package arms it.
func TestContractStoresAreInternallyCoherent(t *testing.T) {
	t.Parallel()

	for _, contract := range tiers.ContractsWithStores() {
		spec, shipped := tiers.ContractStore(contract)
		testkit.True(t, shipped, contract+" answers its own listing")
		testkit.True(t, spec.Store != "", contract+" names a store")

		roles := tiers.ContractRoles(contract)
		testkit.True(t, len(roles) > 0, contract+" carries at least one role op")
		hasTypeArg := false
		for _, role := range roles {
			op, ok := tiers.ContractRoleOp(contract, role)
			testkit.True(t, ok && op != "", contract+"."+role+" delegates to a named op")
			hasTypeArg = hasTypeArg || role == spec.TypeArgRole
		}
		testkit.True(t, hasTypeArg,
			contract+"'s type argument is spoken by one of its own roles")
	}
}

// TestContractRoleOpDeclinesTheUnknown pins the miss arms.
func TestContractRoleOpDeclinesTheUnknown(t *testing.T) {
	t.Parallel()

	_, ok := tiers.ContractRoleOp("lease", "not-a-role")
	testkit.False(t, ok, "an unregistered role names no op")
	_, shipped := tiers.ContractStore("saga")
	testkit.False(t, shipped,
		"a saga needs its steps, which no derivation can mint — the twin floor holds it")
}
