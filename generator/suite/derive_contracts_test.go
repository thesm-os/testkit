// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal for the reason derive_stamps_test.go is: the opener rule
// reads contract roles and partners through the unexported projection
// maps, which only [contractDataOf] over a stamped bag populates.
package suite

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/suite/projection"
)

// openerMethod is the scan Log's producing shape: Scan(ctx) answers a
// handle, and the cursor contract's open role names Close beside it.
// withClose=false drops the partner stamp, the schema-gap arm.
func openerMethod(withClose bool) Method {
	bag := sdk.NewBag()
	shape.MetaContracts.Set(bag, []string{ContractCursor}, "test")
	shape.ContractRoleKey(ContractCursor).Set(bag, ContractCursorOpen, "test")
	if withClose {
		shape.ContractPartnerKey(ContractCursor, ContractCursorClose).Set(bag, "Close", "test")
	}
	roles, partners := contractDataOf(bag)
	return Method{
		Sig: &golang.Sig{
			Name:   "Scan",
			Params: []golang.Param{{Name: "ctx", Source: storefixture.PkgNamed("context", "Context")}},
			Returns: []golang.Return{
				{Source: storefixture.Named("Cursor")},
				{Error: true},
			},
		},
		Contracts:        shape.Contracts(bag),
		contractRoles:    roles,
		contractPartners: partners,
	}
}

func TestOpenerSmokeClosesWhatItOpens(t *testing.T) {
	t.Parallel()

	iface := Iface{Name: "Log", Token: "log", Methods: []Method{openerMethod(true)}}
	plans, refusals := Signature{}.Derive(iface)
	testkit.Len(t, refusals, 0, "the opener shape refuses nothing")
	testkit.Len(t, plans, 1, "no ctx directive: the smoke is the whole signature family")

	p := plans[0]
	scanID, err := p.ID.Render()
	testkit.NoError(t, err, "the derived ID is well formed")
	testkit.Equal(t, scanID, vocab.ID("Scan/smoke"), "the override keeps the smoke's own ID")
	testkit.Equal(t, p.Claim, "Scan survives a call and the cursor it opens closes",
		"the claim is the corpus manifests' spelling")
	testkit.Equal(t, p.Body, projection.Body(projection.SmokeSurvives{
		Call:          projection.CallPlan{Method: "Scan", Args: []projection.Expr{projection.ExprCtx}},
		CloseProduced: "Close",
	}), "the body carries the close partner the contract names")
	testkit.Equal(t, p.Class, vocab.ClassSmoke, "an opener smoke still buckets as a smoke")
}

func TestOpenerWithoutClosePartnerKeepsThePlainSmoke(t *testing.T) {
	t.Parallel()

	iface := Iface{Name: "Log", Token: "log", Methods: []Method{openerMethod(false)}}
	plans, _ := Signature{}.Derive(iface)
	testkit.Len(t, plans, 1, "one smoke either way")
	testkit.Equal(t, plans[0].Claim, "Scan survives a call",
		"partner completeness is the contract schema's to report, not this rule's to guess")
	testkit.Equal(t, plans[0].Body, projection.Body(projection.SmokeSurvives{
		Call: projection.CallPlan{Method: "Scan", Args: []projection.Expr{projection.ExprCtx}},
	}), "no close partner, no close call")
}

// poolRoleMethod stamps one method into the pool contract at a role,
// through the real contract keys.
func poolRoleMethod(sig *golang.Sig, role string, argFields ...string) Method {
	bag := sdk.NewBag()
	shape.MetaContracts.Set(bag, []string{ContractPool}, "test")
	shape.ContractRoleKey(ContractPool).Set(bag, role, "test")
	roles, partners := contractDataOf(bag)
	return Method{
		Sig:              sig,
		ArgFields:        argFields,
		Contracts:        shape.Contracts(bag),
		contractRoles:    roles,
		contractPartners: partners,
	}
}

// poolPair is the borrow shape: Get answers Conn, Put returns it.
func poolPair() []Method {
	ctx := golang.Param{Name: "ctx", Source: storefixture.PkgNamed("context", "Context")}
	get := poolRoleMethod(&golang.Sig{
		Name:    "Get",
		Params:  []golang.Param{ctx},
		Returns: []golang.Return{{Source: storefixture.Named("Conn")}, {Error: true}},
	}, ContractPoolGet)
	put := poolRoleMethod(&golang.Sig{
		Name:    "Put",
		Params:  []golang.Param{ctx, {Name: "c", Source: storefixture.Named("Conn")}},
		Returns: []golang.Return{{Error: true}},
	}, ContractPoolPut, "C")
	return []Method{get, put}
}

func TestBorrowSmokeReturnsWhatItBorrows(t *testing.T) {
	t.Parallel()

	iface := Iface{Name: "Pool", Token: "pool", Methods: poolPair()}
	plans, refusals := Signature{}.Derive(iface)
	testkit.Len(t, refusals, 0, "the produced draw is the borrow's to supply, not a fixture gap")
	testkit.Len(t, plans, 2, "one smoke per method, nothing else without the ctx directive")
	testkit.Equal(t, plans[0].Claim, "Get survives a call", "the producer keeps its plain smoke")

	put := plans[1]
	putID, err := put.ID.Render()
	testkit.NoError(t, err, "the derived ID is well formed")
	testkit.Equal(t, putID, vocab.ID("Put/smoke"), "the override keeps the smoke's own ID")
	testkit.Equal(t, put.Claim, "Put survives returning a borrowed resource",
		"the claim is the corpus manifests' spelling")
	testkit.Equal(t, put.Body, projection.Body(projection.SmokeSurvives{
		Call: projection.CallPlan{
			Method: "Put",
			Args:   []projection.Expr{projection.ExprCtx, projection.ExprBorrowed},
		},
		Borrow: projection.CallPlan{Method: "Get", Args: []projection.Expr{projection.ExprCtx}},
	}), "the body borrows from the get sibling and feeds the borrowed local")
}

func TestPutWithoutAProducerRefusesHonestly(t *testing.T) {
	t.Parallel()

	iface := Iface{Name: "Pool", Token: "pool", Methods: poolPair()[1:]}
	plans, refusals := Signature{}.Derive(iface)
	testkit.Len(t, plans, 0, "nothing to borrow from, nothing the fixture can derive")
	testkit.Len(t, refusals, 1, "the gap is the ordinary undeliverable refusal, named")
	testkit.Equal(t, refusals[0].What, "Put's signature checks", "attributed to the whole family set")
}
