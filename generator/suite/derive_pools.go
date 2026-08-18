// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"slices"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"
	"go.thesmos.sh/testkit/generator/defaults"
	"go.thesmos.sh/testkit/generator/roles"
	"go.thesmos.sh/testkit/generator/suite/projection"
)

// DeriverPools attributes pool refusals. Not in the deriver registry:
// pools are an input projection the shell computes before any check
// derives, not a check family — but a gap in them is still a named
// gap, and it reports under this name.
const DeriverPools DeriverName = "pools"

// poolsOf projects the drawn pools from the roled fields of every
// drawn parameter's struct: one [projection.PoolPlan] per roled
// field, its members derived by the projection's transforms — the
// default stamp verbatim, the distinctness swap, the hostile member.
// Unroled fields pin config values instead and are not this walk's
// concern. Two methods drawing one request struct share its pools.
//
// Refusals, never silence: a roled field with no default, a
// qualified default (a symbol, not a literal the transforms can
// splice), or a member a transform refuses each name the field and
// the consumer action that closes the gap.
func poolsOf(r golang.Resolver, methods []Method) ([]projection.PoolPlan, []Refusal) {
	var pools []projection.PoolPlan
	var refusals []Refusal
	var seen []string

	for _, m := range methods {
		for _, p := range m.CallArgs() {
			decl, resolved := r.Resolve(p.Source)
			s, isStruct := decl.(*sdk.Struct)
			if !resolved || !isStruct || slices.Contains(seen, s.Name) {
				continue
			}
			seen = append(seen, s.Name)
			for _, f := range golang.ExportedFields(s) {
				plan, refusal, roled := poolOf(s.Name, f)
				switch {
				case !roled:
				case refusal != nil:
					refusals = append(refusals, *refusal)
				default:
					pools = append(pools, plan)
				}
			}
		}
	}
	return pools, refusals
}

// poolOf derives one field's pool; roled reports false for a field
// this walk does not own.
func poolOf(structName string, f *sdk.Field) (projection.PoolPlan, *Refusal, bool) {
	role := roles.Of(f.Meta())
	if role == "" {
		return projection.PoolPlan{}, nil, false
	}
	refuse := func(why, remedy string) (projection.PoolPlan, *Refusal, bool) {
		return projection.PoolPlan{}, &Refusal{
			Deriver: DeriverPools,
			What:    "the " + role + " pool from " + structName + "." + f.Name,
			Why:     why,
			Remedy:  remedy,
		}, true
	}

	stamp, stamped := defaults.MetaDefault.Get(f.Meta())
	if !stamped || stamp == "" {
		return refuse("the roled field declares no default, and pool[0] is the default verbatim",
			"declare a //testkit:default beside the role")
	}
	if pkg, _ := defaults.MetaDefaultPkg.Get(f.Meta()); pkg != "" {
		return refuse("a qualified default names a symbol, not a literal the member transforms can splice",
			"spell the default as a literal, or supply the pool through the config")
	}
	distinct, ok := projection.DistinctMember(projection.Expr(stamp))
	if !ok {
		return refuse("the default carries no distinctness swap point, and two equal members fund no miss",
			"spell the default's textual payload test-*, or supply the pool through the config")
	}
	hostile, ok := projection.HostileMember(projection.Expr(stamp), role)
	if !ok {
		return refuse("no hostile member derives from the default's shape",
			"supply the pool through the config, hostile member included")
	}
	return projection.PoolPlan{
		Role:    role,
		Field:   projection.PoolFieldName(f.Name),
		Members: [3]projection.Expr{projection.Expr(stamp), distinct, hostile},
	}, nil, true
}
