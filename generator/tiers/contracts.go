// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers

import "slices"

// ContractStore returns the [engine/model/ref] store the named contract's
// roles delegate to, and whether one ships.
//
// A contract row exists only where the oracle is derivable whole: every role
// in the op table resolvable from the stamps, the store's type argument
// spoken by a role's own signature, and every constructor argument either a
// sentinel this generator can mint or a semantics choice it can make. The
// families that fail that bar — a pool needing a resource constructor, a
// saga needing its steps, a coalescer needing the function it coalesces —
// stay on the twin floor, whose header says so, and `ref=` raises them.
func ContractStore(contract string) (ContractStoreSpec, bool) {
	spec, shipped := contractStores[contract]
	return spec, shipped
}

// ContractStoreSpec is one derivable contract oracle.
type ContractStoreSpec struct {
	// Store is the ref type; "New" + Store its constructor — the naming
	// convention the shape oracles already rely on.
	Store string

	// TypeArgRole names the role whose signature speaks the store's one
	// type argument: its first parameter, or its first result when
	// TypeArgResult is set.
	TypeArgRole   string
	TypeArgResult bool

	// Errs are the constructor's error arguments in declaration order. A
	// named entry mints a sentinel; an empty one renders nil — the oracle's
	// lenient arm, chosen where two legitimate dialects exist and the
	// stricter one would fail the weaker. The corpus proved the lease row:
	// releasing what was never held is ordinary Go to its subject, and a
	// strict oracle read the no-op as divergence.
	Errs []ContractErr
}

// ContractErr is one constructor error argument. NilUnder names a mixin
// whose presence on the Role method renders nil in the sentinel's place —
// the claim says the stricter dialect does not apply, and the oracle's nil
// arm is the lenient one. The corpus proved the row both ways: a strict
// lease refuses the second acquire, and an idempotent one re-enters.
type ContractErr struct {
	Suffix, Msg    string
	Role, NilUnder string
}

// ContractRoleOp returns the oracle method the named contract role delegates
// to.
func ContractRoleOp(contract, role string) (string, bool) {
	op, ok := contractRoleOps[contract+"."+role]
	return op, ok
}

// ContractRoles returns the named contract's role vocabulary, sorted — the
// set an interface must resolve completely for the oracle to derive.
func ContractRoles(contract string) []string {
	prefix := contract + "."
	out := make([]string, 0, 2)
	for key := range contractRoleOps {
		if rest, matched := trimPrefix(key, prefix); matched {
			out = append(out, rest)
		}
	}
	slices.Sort(out)
	return out
}

// ContractsWithStores returns every contract carrying a store row, sorted,
// for the censuses.
func ContractsWithStores() []string {
	out := make([]string, 0, len(contractStores))
	for name := range contractStores {
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}

// trimPrefix is strings.CutPrefix without the import for two call sites.
func trimPrefix(s, prefix string) (string, bool) {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):], true
	}
	return "", false
}

// The lease vocabulary, spelled once: the role names are the directive's
// and the op names the tracker's.
const (
	roleAcquire = "acquire"
	opAcquire   = "Acquire"
	opRelease   = "Release"
)

// The contract oracle tables.
//
//nolint:gochecknoglobals // lookup tables, read-only after init.
var (
	contractStores = map[string]ContractStoreSpec{
		contractLease: {
			Store:       "LeaseTracker",
			TypeArgRole: roleAcquire,
			Errs: []ContractErr{
				{
					Suffix: "Held", Msg: "the model reference already holds the key",
					Role: roleAcquire, NilUnder: mixinIdempotent,
				},
				// Lenient release: giving up what was never taken is
				// ordinary Go to the corpus subject, and nil is the
				// tracker's spelling of that dialect.
				{},
			},
		},
	}
	contractRoleOps = map[string]string{
		contractLease + ".acquire": opAcquire,
		contractLease + ".release": opRelease,
	}
)
