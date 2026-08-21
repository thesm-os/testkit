// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/suite"
	"go.thesmos.sh/testkit/generator/tiers"
)

// TestContractShapeArms pins the contract-shape closure family: a
// handle-answering begin, the terminal pair that threads it, the saga's
// coordinating run, the compute-taking call, the failing-body run and the
// page-shaped walk — each binding at its fixture's types or refusing with
// what is missing.
func TestContractShapeArms(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{Subject: suite.Subject{IfaceName: "Contract"}}

	t.Run("a handle-answering begin binds and a flat one refuses", func(t *testing.T) {
		t.Parallel()
		field, reason := bindField(b, lawid.TwoPhaseMutex, "Begin",
			projected("Begin", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(namedRef("Tx")), errRet}))
		testkit.True(t, reason == "" && field.Out != nil, "the handle draw binds: "+reason)

		_, reason = bindField(b, lawid.TwoPhaseMutex, "Begin",
			projected("Begin", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("answers no handle", "a flat begin threads nothing")

		_, reason = bindField(b, lawid.TwoPhaseMutex, "Begin",
			projected("Begin", []golang.Param{arg("ctx", ctxRef()), arg("opts", namedRef(qStr))},
				[]golang.Return{res(namedRef("Tx")), errRet}))
		testkit.Assert(t, reason).Contains("no handle draw supplies", "an input nothing draws")
	})

	// The staging write a mid-transaction claim is stated about. Bound from
	// the interface rather than supplied as a closure, because a supplied one
	// reaches past to the concrete store and no defect worn on the interface
	// can then reach the law — bound, green, and unfalsifiable.
	t.Run("a staging write threads the handle beside a key and a value", func(t *testing.T) {
		t.Parallel()
		staging := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}

		field, reason := bindField(staging, lawid.TransactionNoMidTxVisibility, "TxPut",
			projected("Put", []golang.Param{
				arg("ctx", ctxRef()), arg("h", namedRef("Tx")),
				arg("k", namedRef(qStr)), arg("v", namedRef("Value")),
			}, []golang.Return{errRet}))
		testkit.True(t, reason == "", "the staging write binds: "+reason)
		testkit.True(t, field != nil && field.In != nil && field.Key != nil && field.Value != nil,
			"and carries the handle, the key and the value it writes")

		_, reason = bindField(staging, lawid.TransactionNoMidTxVisibility, "TxPut",
			projected("Put", []golang.Param{
				arg("ctx", ctxRef()), arg("k", namedRef(qStr)), arg("v", namedRef("Value")),
			}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("inside an open handle",
			"a keyed write with no handle stages nothing a transaction owns")
	})

	t.Run("a terminal operation threads begin's handle", func(t *testing.T) {
		t.Parallel()
		begin := projected("Begin", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("Tx")), errRet})
		shape.ContractRoleKey("tx").Set(begin.Source.EnsureMeta(), "begin", "test")
		shape.ContractPartnerKey("tx", "commit").Set(begin.Source.EnsureMeta(), "Commit", "test")
		commit := projected("Commit",
			[]golang.Param{arg("ctx", ctxRef()), arg("h", namedRef("Tx"))}, []golang.Return{errRet})
		r := tiers.Rule{Law: lawid.TwoPhaseMutex, Fields: []tiers.Field{
			{Name: "Begin", Kind: tiers.KindRole, From: "tx.begin"},
			{Name: "Commit", Kind: tiers.KindRole, From: "tx.commit"},
		}}
		field, reason := lawFieldOf(b, harnessOf(begin, commit), r, r.Fields[1], begin, nil)
		testkit.True(t, reason == "" && field.In != nil, "the terminal op binds: "+reason)

		stranger := projected("Commit",
			[]golang.Param{arg("ctx", ctxRef()), arg("h", namedRef("Other"))}, []golang.Return{errRet})
		_, reason = lawFieldOf(b, harnessOf(begin, stranger), r, r.Fields[1], begin, nil)
		testkit.Assert(t, reason).Contains("settles a", "a stranger handle is named, not compiled against")
	})

	t.Run("the saga run closes over drawn steps and its compensation", func(t *testing.T) {
		t.Parallel()
		step := projected("Step",
			[]golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))}, []golang.Return{errRet})
		shape.ContractRoleKey("saga").Set(step.Source.EnsureMeta(), "step", "test")
		shape.ContractPartnerKey("saga", "compensate").Set(step.Source.EnsureMeta(), "Compensate", "test")
		comp := projected("Compensate",
			[]golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))}, []golang.Return{errRet})
		r := tiers.Rule{Law: lawid.SagaFullCompensation, Fields: []tiers.Field{
			{Name: "Run", Kind: tiers.KindRole, From: "saga.step"},
		}}

		pooled := &Bindings{
			Subject: suite.Subject{IfaceName: "Contract"},
			Values:  Pool{Type: sdk.Builtin("Value"), Q: "Value"},
			Actions: []*Action{{Method: "Step", Pool: poolValues}},
		}
		field, reason := lawFieldOf(pooled, harnessOf(step, comp), r, r.Fields[0], step, nil)
		testkit.True(t, reason == "", "the coordinating run binds: "+reason)
		testkit.Equal(t, field.Partner, "Compensate", "unwinding through the pinned pairing")
		testkit.Equal(t, field.Pool, poolValues, "over the drawn steps")

		_, reason = lawFieldOf(b, harnessOf(step, comp), r, r.Fields[0], step, nil)
		testkit.Assert(t, reason).Contains("no action here declares", "no pool, no steps to draw")
	})

	t.Run("a compute-taking call binds and a computeless one refuses", func(t *testing.T) {
		t.Parallel()
		field, reason := bindField(b, lawid.SingleflightCoalesces, fCall,
			projected("Run",
				[]golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr)), arg("compute", funcRef())},
				[]golang.Return{res(namedRef(qStr)), errRet}))
		testkit.True(t, reason == "" && field.Key != nil && field.Out != nil,
			"the deduplicated call binds: "+reason)

		_, reason = bindField(b, lawid.SingleflightCoalesces, fCall,
			projected("Run", []golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr))},
				[]golang.Return{res(namedRef(qStr)), errRet}))
		testkit.Assert(t, reason).Contains("no compute to deduplicate", "nothing to count")
	})

	t.Run("a failing-body run binds and a bodyless one refuses", func(t *testing.T) {
		t.Parallel()
		field, reason := bindField(b, lawid.TransactionRollback, "Run",
			projected("Run", []golang.Param{arg("ctx", ctxRef()), arg("body", funcRef())},
				[]golang.Return{errRet}))
		testkit.True(t, reason == "" && field != nil, "the scoped run binds: "+reason)

		_, reason = bindField(b, lawid.TransactionRollback, "Run",
			projected("Run", []golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr))},
				[]golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("accepts no failing body", "a keyed run cannot be made to fail")
	})

	t.Run("a page-shaped read binds and each missing half refuses", func(t *testing.T) {
		t.Parallel()
		page := func(next, more *node.TypeRef) *suite.Method {
			return projected("Page",
				[]golang.Param{arg("ctx", ctxRef()), arg("cur", namedRef("Cursor"))},
				[]golang.Return{res(sliceRef(namedRef("Value"))), res(next), res(more), errRet})
		}
		field, reason := bindField(b, lawid.PaginatorNoDuplicates, "Page",
			page(namedRef("Cursor"), namedRef("bool")))
		testkit.True(t, reason == "" && field.In != nil && field.Out != nil,
			"the page walk binds: "+reason)

		_, reason = bindField(b, lawid.PaginatorNoDuplicates, "Page",
			projected("Page", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
				[]golang.Return{res(namedRef("Value")), errRet}))
		testkit.Assert(t, reason).Contains("no cursor to resume from", "a keyed read walks nothing")

		_, reason = bindField(b, lawid.PaginatorNoDuplicates, "Page",
			page(namedRef(qStr), namedRef("bool")))
		testkit.Assert(t, reason).Contains("which is not its", "the next cursor resumes at another type")

		_, reason = bindField(b, lawid.PaginatorNoDuplicates, "Page",
			page(namedRef("Cursor"), namedRef("int")))
		testkit.Assert(t, reason).Contains("never says whether more remains", "no termination signal")
	})
}

// TestContractShapeRefusalClauses walks the family's remaining refusal
// clauses one by one — each is a distinct wrong fixture a named refusal must
// meet, and a clause nothing exercises is a message nothing proofread.
func TestContractShapeRefusalClauses(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{Subject: suite.Subject{IfaceName: "Contract"}}

	t.Run("a terminal pair's own wrong shapes refuse", func(t *testing.T) {
		t.Parallel()
		begin := projected("Begin", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("Tx")), errRet})
		shape.ContractRoleKey("tx").Set(begin.Source.EnsureMeta(), "begin", "test")
		shape.ContractPartnerKey("tx", "commit").Set(begin.Source.EnsureMeta(), "Commit", "test")
		r := tiers.Rule{Law: lawid.TwoPhaseMutex, Fields: []tiers.Field{
			{Name: fBegin, Kind: tiers.KindRole, From: "tx.begin"},
			{Name: fCommit, Kind: tiers.KindRole, From: "tx.commit"},
		}}

		wide := projected("Commit",
			[]golang.Param{arg("ctx", ctxRef()), arg("h", namedRef("Tx")), arg("note", namedRef(qStr))},
			[]golang.Return{errRet})
		_, reason := lawFieldOf(b, harnessOf(begin, wide), r, r.Fields[1], begin, nil)
		testkit.Assert(t, reason).Contains("does not settle one handle", "two inputs settle nothing")

		flatBegin := projected("Begin", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet})
		shape.ContractRoleKey("tx").Set(flatBegin.Source.EnsureMeta(), "begin", "test")
		shape.ContractPartnerKey("tx", "commit").Set(flatBegin.Source.EnsureMeta(), "Commit", "test")
		commit := projected("Commit",
			[]golang.Param{arg("ctx", ctxRef()), arg("h", namedRef("Tx"))}, []golang.Return{errRet})
		_, reason = lawFieldOf(b, harnessOf(flatBegin, commit), r, r.Fields[1], flatBegin, nil)
		testkit.Assert(t, reason).Contains("answers none", "no handle to thread")
	})

	t.Run("the saga run's own wrong shapes refuse", func(t *testing.T) {
		t.Parallel()
		r := tiers.Rule{Law: lawid.SagaFullCompensation, Fields: []tiers.Field{
			{Name: fRun, Kind: tiers.KindRole, From: "saga.step"},
		}}
		pooled := &Bindings{
			Subject: suite.Subject{IfaceName: "Contract"},
			Values:  Pool{Type: sdk.Builtin("Value"), Q: "Value"},
			Actions: []*Action{{Method: "Step", Pool: poolValues}},
		}

		wide := projected("Step",
			[]golang.Param{arg("ctx", ctxRef()), arg("a", namedRef("Value")), arg("b", namedRef("Value"))},
			[]golang.Return{errRet})
		shape.ContractRoleKey("saga").Set(wide.Source.EnsureMeta(), "step", "test")
		_, reason := lawFieldOf(pooled, harnessOf(wide), r, r.Fields[0], wide, nil)
		testkit.Assert(t, reason).Contains("does not step one value", "two inputs step nothing")

		stranger := stamp(projected("Step",
			[]golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Other"))},
			[]golang.Return{errRet}), "", "", "Other")
		shape.ContractRoleKey("saga").Set(stranger.Source.EnsureMeta(), "step", "test")
		_, reason = lawFieldOf(pooled, harnessOf(stranger), r, r.Fields[0], stranger, nil)
		testkit.Assert(t, reason).Contains("beside a pool of", "steps a type the pool never draws")

		orphan := projected("Step",
			[]golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))}, []golang.Return{errRet})
		shape.ContractRoleKey("saga").Set(orphan.Source.EnsureMeta(), "step", "test")
		_, reason = lawFieldOf(pooled, harnessOf(orphan), r, r.Fields[0], orphan, nil)
		testkit.Assert(t, reason).Contains("does not stamp", "no compensation, no unwinding")
	})

	t.Run("a compute of the wrong kind refuses", func(t *testing.T) {
		t.Parallel()
		_, reason := bindField(b, lawid.SingleflightCoalesces, fCall,
			projected("Run",
				[]golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr)), arg("note", namedRef(qStr))},
				[]golang.Return{res(namedRef(qStr)), errRet}))
		testkit.Assert(t, reason).Contains("no compute to deduplicate", "a second string is not a callable")

		_, reason = bindField(b, lawid.SingleflightCoalesces, fCall,
			projected("Run",
				[]golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr)), arg("compute", funcRef())},
				[]golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("nothing to observe", "a call answering nothing shares nothing")
	})

	t.Run("a body-run of the wrong arity refuses", func(t *testing.T) {
		t.Parallel()
		_, reason := bindField(b, lawid.TransactionRollback, fRun,
			projected("Run",
				[]golang.Param{arg("ctx", ctxRef()), arg("body", funcRef()), arg("note", namedRef(qStr))},
				[]golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("accepts no failing body", "two inputs scope nothing")
	})

	t.Run("a page walk with no error or no cursor refuses", func(t *testing.T) {
		t.Parallel()
		_, reason := bindField(b, lawid.PaginatorNoDuplicates, fPage,
			projected("Page",
				[]golang.Param{arg("ctx", ctxRef()), arg("cur", namedRef("Cursor"))},
				[]golang.Return{
					res(sliceRef(namedRef("Value"))), res(namedRef("Cursor")), res(namedRef("bool")),
				}))
		testkit.Assert(t, reason).Contains("no cursor to resume from", "three results are not a page")

		_, reason = bindField(b, lawid.PaginatorNoDuplicates, fPage,
			projected("Page", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{
					res(sliceRef(namedRef("Value"))), res(namedRef("Cursor")), res(namedRef("bool")), errRet,
				}))
		testkit.Assert(t, reason).Contains("no cursor to resume from", "nothing to resume by")
	})

	t.Run("a probe over an answerless call refuses", func(t *testing.T) {
		t.Parallel()
		bare := projected("Run",
			[]golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr)), arg("compute", funcRef())},
			[]golang.Return{errRet})
		r := tiers.Rule{Law: lawid.SingleflightCoalesces, Fields: []tiers.Field{
			{Name: fCall, Kind: tiers.KindRole, From: fromSelf},
			{Name: "Compute", Kind: tiers.KindHandle, From: handleCoalesce},
		}}
		_, reason := lawFieldOf(b, nil, r, r.Fields[1], bare, nil)
		testkit.Assert(t, reason).Contains("nothing to observe", "no answer, nothing for the probe to type")
	})
}

// TestContractShapeAnswerClauses covers the family's answer-shaped
// refusals: a begin answering two things threads neither, and a body-run
// answering more than an error is not the scope the law forwards into.
func TestContractShapeAnswerClauses(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{Subject: suite.Subject{IfaceName: "Contract"}}

	_, reason := bindField(b, lawid.TwoPhaseMutex, fBegin,
		projected("Begin", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("Tx")), res(namedRef("Lease")), errRet}))
	testkit.Assert(t, reason).Contains("answers no handle", "two answers thread neither")

	_, reason = bindField(b, lawid.TransactionRollback, fRun,
		projected("Run", []golang.Param{arg("ctx", ctxRef()), arg("body", funcRef())},
			[]golang.Return{res(namedRef(qStr)), errRet}))
	testkit.Assert(t, reason).Contains("accepts no failing body", "an answering run is another shape")
}

// TestReplicaClosureShapes pins the multi-replica family: the pairwise sync
// composed into the star round, the per-replica settle, and the refusals a
// wrong peer or a wide settle earns.
func TestReplicaClosureShapes(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}

	syncRole := projected("Sync",
		[]golang.Param{arg("ctx", ctxRef()), arg("peer", namedRef("Mixed"))}, []golang.Return{errRet})
	field, reason := bindField(b, lawid.EventualConvergence, "Sync", syncRole)
	testkit.True(t, reason == "" && field != nil, "the pairwise sync binds: "+reason)

	stranger := projected("Sync",
		[]golang.Param{arg("ctx", ctxRef()), arg("peer", namedRef("Other"))}, []golang.Return{errRet})
	_, reason = bindField(b, lawid.EventualConvergence, "Sync", stranger)
	testkit.Assert(t, reason).Contains("syncs with a Other", "a foreign peer is named, not guessed")

	wide := projected("Sync",
		[]golang.Param{arg("ctx", ctxRef()), arg("a", namedRef("Mixed")), arg("b", namedRef("Mixed"))},
		[]golang.Return{errRet})
	_, reason = bindField(b, lawid.EventualConvergence, "Sync", wide)
	testkit.Assert(t, reason).Contains("does not sync with one peer", "two peers are another protocol")

	settle := projected("Settle", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet})
	field, reason = bindField(b, lawid.EventualConvergence, "Settle", settle)
	testkit.True(t, reason == "" && field != nil, "the per-replica settle binds: "+reason)

	wideSettle := projected("Settle",
		[]golang.Param{arg("ctx", ctxRef()), arg("n", namedRef("int"))}, []golang.Return{errRet})
	_, reason = bindField(b, lawid.EventualConvergence, "Settle", wideSettle)
	testkit.Assert(t, reason).Contains("not a nullary settle", "a parameterised settle draws nothing")
}

// TestMidTxDoorAndReadbackPool pins the mid-transaction wiring: the TxPut
// door spelled at the handle, key and read-back types, the readback pool at
// the observed reader's answer, and the refusals each earns without its
// anchors.
func TestMidTxStagingAndReadbackPool(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	begin := projected("Begin", []golang.Param{arg("ctx", ctxRef())},
		[]golang.Return{res(namedRef("Tx")), errRet})
	shape.ContractRoleKey("tx").Set(begin.Source.EnsureMeta(), "begin", "test")
	get := stamp(projected("Get",
		[]golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr))},
		[]golang.Return{res(namedRef("Value")), errRet}), "reader", qStr, "Value")
	// The staging write is derived from the interface now, not supplied. A
	// door reaching past to the concrete store is what made this law
	// unfalsifiable: every mutant the prover wore failed the type assertion
	// inside the consumer's closure before the law observed anything.
	put := projected("Put", []golang.Param{
		arg("ctx", ctxRef()), arg("h", namedRef("Tx")),
		arg("key", namedRef(qStr)), arg("v", namedRef("Value")),
	}, []golang.Return{errRet})
	r := tiers.Rule{Law: lawid.TransactionNoMidTxVisibility, Fields: []tiers.Field{
		{Name: fBegin, Kind: tiers.KindRole, From: "tx.begin"},
		{Name: fTxPut, Kind: tiers.KindRole, From: "family.handlewriter"},
		{Name: fRead, Kind: tiers.KindRole, From: "family.reader"},
		{Name: "Values", Kind: tiers.KindGenerator, From: "readback"},
	}}

	t.Run("the staging write and the pool bind at the fixture's types", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{
			Subject: suite.Subject{IfaceName: "Contract"},
			Keys:    Pool{Type: sdk.Builtin(qStr), Q: qStr, Field: "Key"},
		}
		field, reason := lawFieldOf(b, harnessOf(begin, put, get), r, r.Fields[1], begin, get)
		testkit.True(t, reason == "", "the staging write binds: "+reason)
		testkit.Equal(t, string(field.Kind()), "model.lawfield.HandleWrite",
			"as a closure over a real method, not a door")

		field, reason = lawFieldOf(b, harnessOf(begin, put, get), r, r.Fields[3], begin, get)
		testkit.True(t, reason == "", "the readback pool binds: "+reason)
		testkit.Equal(t, field.Pool, "readback", "at the observed reader's answer")
	})

	t.Run("each anchor's absence refuses by name", func(t *testing.T) {
		t.Parallel()
		pooled := &Bindings{
			Subject: suite.Subject{IfaceName: "Contract"},
			Keys:    Pool{Type: sdk.Builtin(qStr), Q: qStr, Field: "Key"},
		}
		_, reason := lawFieldOf(pooled, harnessOf(begin, get), r, r.Fields[1], begin, get)
		testkit.Assert(t, reason).Contains("no write threading an open handle",
			"nothing on the interface stages, so the claim declines")

		_, reason = lawFieldOf(pooled, harnessOf(begin), r, r.Fields[3], begin, nil)
		testkit.Assert(t, reason).Contains("no keyed reader", "no read-back, no domain to draw")
	})
}

// TestStampedSentinel pins the declared-sentinel resolution: a contract error
// row naming a stamped parameter hands the oracle the declaration's own error
// identity, and every other shape of the stamp falls back to minting.
func TestStampedSentinel(t *testing.T) {
	t.Parallel()

	carrier := &suite.Method{Sig: &golang.Sig{Source: &node.Method{}}}
	carrier.Contracts = []string{"lease"}

	t.Run("an unnamed parameter mints", func(t *testing.T) {
		t.Parallel()
		_, stamped := stampedSentinel(nil, carrier, "lease", "")
		testkit.False(t, stamped, "no parameter, no stamp to read")
	})

	t.Run("an unstamped parameter mints", func(t *testing.T) {
		t.Parallel()
		_, stamped := stampedSentinel(nil, carrier, "lease", "held")
		testkit.False(t, stamped, "the declaration said nothing")
	})

	t.Run("an unqualified stamp mints", func(t *testing.T) {
		t.Parallel()
		bare := &suite.Method{Sig: &golang.Sig{Source: &node.Method{}}}
		bare.Contracts = []string{"lease"}
		shape.ContractParamKey("lease", "held").Set(bare.Source.EnsureMeta(), "ErrHeld", "test")
		_, stamped := stampedSentinel(nil, bare, "lease", "held")
		testkit.False(t, stamped, "a bare name carries no package to import")
	})

	t.Run("a qualified stamp is the oracle's sentinel", func(t *testing.T) {
		t.Parallel()
		host := &suite.Method{Sig: &golang.Sig{Source: &node.Method{}}}
		host.Contracts = []string{"lease"}
		shape.ContractParamKey("lease", "held").
			Set(host.Source.EnsureMeta(), "example.com/lease.ErrHeld", "test")
		sym, stamped := stampedSentinel(nil, host, "lease", "held")
		testkit.True(t, stamped && sym != nil, "one identity for the oracle and the law")
	})
}
